package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"math/rand/v2"
	"time"

	"net/http"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/zachbroad/nitrohook/internal/dispatch"
	"github.com/zachbroad/nitrohook/internal/metrics"
	"github.com/zachbroad/nitrohook/internal/model"
	"github.com/zachbroad/nitrohook/internal/script"
	"github.com/zachbroad/nitrohook/internal/store"
)

const (
	streamName    = "deliveries"
	consumerGroup = "fanout-workers"
)

type FanoutWorker struct {
	store          *store.Store
	rdb            *redis.Client
	concurrency    int
	maxRetries     int
	retryBaseDelay time.Duration
	pollInterval   time.Duration
}

func New(s *store.Store, rdb *redis.Client, concurrency, maxRetries int, retryBaseDelay, deliveryTimeout, pollInterval time.Duration) *FanoutWorker {
	// Register dispatchers
	httpClient := &http.Client{Timeout: deliveryTimeout}
	dispatch.Register(model.ActionTypeWebhook, &dispatch.WebhookDispatcher{Client: httpClient})
	dispatch.Register(model.ActionTypeJavascript, &dispatch.JavascriptDispatcher{})
	dispatch.Register(model.ActionTypeSlack, &dispatch.SlackDispatcher{Client: httpClient})
	dispatch.Register(model.ActionTypeSMTP, &dispatch.SMTPDispatcher{})
	dispatch.Register(model.ActionTypeTwilio, &dispatch.TwilioDispatcher{Client: httpClient})

	return &FanoutWorker{
		store:          s,
		rdb:            rdb,
		concurrency:    concurrency,
		maxRetries:     maxRetries,
		retryBaseDelay: retryBaseDelay,
		pollInterval:   pollInterval,
	}
}

func (w *FanoutWorker) Start(ctx context.Context) error {
	err := w.rdb.XGroupCreateMkStream(ctx, streamName, consumerGroup, "0").Err()
	if err != nil && err.Error() != "BUSYGROUP Consumer Group name already exists" {
		return fmt.Errorf("create consumer group: %w", err)
	}

	for i := range w.concurrency {
		consumer := fmt.Sprintf("worker-%d", i)
		go w.consumeStream(ctx, consumer)
	}

	go w.pollPending(ctx)
	go w.pollRetries(ctx)

	return nil
}

func (w *FanoutWorker) consumeStream(ctx context.Context, consumer string) {
	for {
		if ctx.Err() != nil {
			return
		}

		streams, err := w.rdb.XReadGroup(ctx, &redis.XReadGroupArgs{
			Group:    consumerGroup,
			Consumer: consumer,
			Streams:  []string{streamName, ">"},
			Count:    1,
			Block:    5 * time.Second,
		}).Result()
		if err != nil {
			if err == redis.Nil || ctx.Err() != nil {
				continue
			}
			slog.Error("xreadgroup error", "error", err, "consumer", consumer)
			time.Sleep(time.Second)
			continue
		}

		for _, stream := range streams {
			for _, msg := range stream.Messages {
				deliveryIDStr, ok := msg.Values["delivery_id"].(string)
				if !ok {
					slog.Error("invalid delivery_id in stream message", "msg_id", msg.ID)
					w.rdb.XAck(ctx, streamName, consumerGroup, msg.ID)
					continue
				}

				deliveryID, err := uuid.Parse(deliveryIDStr)
				if err != nil {
					slog.Error("failed to parse delivery_id", "error", err, "value", deliveryIDStr)
					w.rdb.XAck(ctx, streamName, consumerGroup, msg.ID)
					continue
				}

				force := msg.Values["force"] == "1"
				w.processDelivery(ctx, deliveryID, force)
				w.rdb.XAck(ctx, streamName, consumerGroup, msg.ID)
				w.rdb.XDel(ctx, streamName, msg.ID)
			}
		}
	}
}

func (w *FanoutWorker) processDelivery(ctx context.Context, deliveryID uuid.UUID, force bool) {
	delivery, err := w.store.Deliveries.GetByID(ctx, deliveryID)
	if err != nil {
		slog.Error("failed to get delivery", "error", err, "delivery_id", deliveryID)
		return
	}

	if force {
		if delivery.Status != model.DeliveryPending && delivery.Status != model.DeliveryRecorded {
			return
		}
	} else {
		if delivery.Status != model.DeliveryPending {
			return
		}
	}

	src, err := w.store.Sources.GetByID(ctx, delivery.SourceID)
	if err != nil {
		slog.Error("failed to get source for delivery", "error", err, "delivery_id", deliveryID)
		return
	}

	if !force && src.Mode == "record" {
		w.store.Deliveries.UpdateStatus(ctx, deliveryID, model.DeliveryRecorded)
		return
	}

	if err := w.store.Deliveries.UpdateStatus(ctx, deliveryID, model.DeliveryProcessing); err != nil {
		slog.Error("failed to update delivery status", "error", err, "delivery_id", deliveryID)
		return
	}

	actions, err := w.store.Actions.ListActiveBySource(ctx, delivery.SourceID)
	if err != nil {
		slog.Error("failed to list actions", "error", err, "delivery_id", deliveryID)
		return
	}

	if len(actions) == 0 {
		w.store.Deliveries.UpdateStatus(ctx, deliveryID, model.DeliveryCompleted)
		return
	}

	payload := delivery.Payload
	headers := delivery.Headers
	activeActions := actions

	if src.ScriptBody != nil && *src.ScriptBody != "" {
		transformResult, err := w.runTransform(*src.ScriptBody, delivery, actions)
		if err != nil {
			slog.Error("script execution failed", "error", err, "delivery_id", deliveryID)
			w.store.Deliveries.UpdateStatus(ctx, deliveryID, model.DeliveryFailed)
			return
		}

		if transformResult.Dropped {
			slog.Info("script dropped delivery", "delivery_id", deliveryID)
			w.store.Deliveries.UpdateStatus(ctx, deliveryID, model.DeliveryCompleted)
			return
		}

		transformedPayload, err := json.Marshal(transformResult.Payload)
		if err != nil {
			slog.Error("failed to marshal transformed payload", "error", err, "delivery_id", deliveryID)
			w.store.Deliveries.UpdateStatus(ctx, deliveryID, model.DeliveryFailed)
			return
		}
		transformedHeaders, err := json.Marshal(transformResult.Headers)
		if err != nil {
			slog.Error("failed to marshal transformed headers", "error", err, "delivery_id", deliveryID)
			w.store.Deliveries.UpdateStatus(ctx, deliveryID, model.DeliveryFailed)
			return
		}

		if err := w.store.Deliveries.SetTransformed(ctx, deliveryID, transformedPayload, transformedHeaders); err != nil {
			slog.Error("failed to persist transformed data", "error", err, "delivery_id", deliveryID)
		}

		payload = transformedPayload
		headers = transformedHeaders

		if len(transformResult.Actions) > 0 {
			activeActions = filterActions(actions, transformResult.Actions)
		} else {
			w.store.Deliveries.UpdateStatus(ctx, deliveryID, model.DeliveryCompleted)
			return
		}
	}

	if len(activeActions) == 0 {
		w.store.Deliveries.UpdateStatus(ctx, deliveryID, model.DeliveryCompleted)
		return
	}

	for _, action := range activeActions {
		p, h, skipped := w.applyActionTransform(&action, payload, headers)
		if skipped {
			slog.Info("action transform skipped action", "delivery_id", deliveryID, "action_id", action.ID)
			continue
		}
		w.dispatchAction(ctx, delivery, &action, 1, p, h)
	}

	w.rollUpDeliveryStatus(ctx, deliveryID)
}

func (w *FanoutWorker) dispatchAction(ctx context.Context, delivery *model.Delivery, action *model.Action, attemptNumber int, payload, headers json.RawMessage) bool {
	attempt, err := w.store.Deliveries.CreateAttempt(ctx, delivery.ID, action.ID, attemptNumber)
	if err != nil {
		slog.Error("failed to create attempt", "error", err)
		return false
	}

	d, err := dispatch.Get(action.Type)
	if err != nil {
		errMsg := err.Error()
		w.store.Deliveries.UpdateAttempt(ctx, attempt.ID, model.AttemptFailed, nil, nil, &errMsg, nil)
		return false
	}

	dispatchStart := time.Now()
	result := d.Dispatch(ctx, action, delivery.ID.String(), payload, headers)
	metrics.DispatchDuration.Observe(time.Since(dispatchStart).Seconds())

	if result.Success {
		w.store.Deliveries.UpdateAttempt(ctx, attempt.ID, model.AttemptSuccess, result.ResponseStatus, result.ResponseBody, nil, nil)
		metrics.DeliveriesDispatched.WithLabelValues(string(action.Type), "success").Inc()
		return true
	}

	nextRetry := w.nextRetryTime(attemptNumber)
	w.store.Deliveries.UpdateAttempt(ctx, attempt.ID, model.AttemptFailed, result.ResponseStatus, result.ResponseBody, result.ErrorMessage, nextRetry)
	metrics.DeliveriesDispatched.WithLabelValues(string(action.Type), "failed").Inc()
	return false
}

func (w *FanoutWorker) dispatchToAction(ctx context.Context, delivery *model.Delivery, action *model.Action, attemptNumber int) bool {
	payload := delivery.Payload
	headers := delivery.Headers
	if delivery.TransformedPayload != nil {
		payload = delivery.TransformedPayload
	}
	if delivery.TransformedHeaders != nil {
		headers = delivery.TransformedHeaders
	}
	payload, headers, skipped := w.applyActionTransform(action, payload, headers)
	if skipped {
		slog.Info("action transform skipped action on retry", "delivery_id", delivery.ID, "action_id", action.ID)
		return true
	}
	return w.dispatchAction(ctx, delivery, action, attemptNumber, payload, headers)
}

func (w *FanoutWorker) runTransform(scriptBody string, delivery *model.Delivery, actions []model.Action) (*script.TransformResult, error) {
	var payloadMap map[string]any
	if err := json.Unmarshal(delivery.Payload, &payloadMap); err != nil {
		return nil, fmt.Errorf("unmarshal payload: %w", err)
	}

	var headersMap map[string]string
	if err := json.Unmarshal(delivery.Headers, &headersMap); err != nil {
		return nil, fmt.Errorf("unmarshal headers: %w", err)
	}

	actionRefs := make([]script.ActionRef, len(actions))
	for i, a := range actions {
		targetURL := ""
		if a.TargetURL != nil {
			targetURL = *a.TargetURL
		}
		actionRefs[i] = script.ActionRef{ID: a.ID, TargetURL: targetURL}
	}

	input := script.TransformInput{
		Payload: payloadMap,
		Headers: headersMap,
		Actions: actionRefs,
	}

	return script.Run(scriptBody, input)
}

func (w *FanoutWorker) applyActionTransform(action *model.Action, payload, headers json.RawMessage) (json.RawMessage, json.RawMessage, bool) {
	if action.TransformScript == nil || *action.TransformScript == "" {
		return payload, headers, false
	}

	var payloadMap map[string]any
	if err := json.Unmarshal(payload, &payloadMap); err != nil {
		slog.Error("action transform: failed to unmarshal payload, using original", "error", err, "action_id", action.ID)
		return payload, headers, false
	}

	var headersMap map[string]string
	if err := json.Unmarshal(headers, &headersMap); err != nil {
		slog.Error("action transform: failed to unmarshal headers, using original", "error", err, "action_id", action.ID)
		return payload, headers, false
	}

	result, err := script.RunActionTransform(*action.TransformScript, payloadMap, headersMap)
	if err != nil {
		slog.Error("action transform: script failed, using original payload", "error", err, "action_id", action.ID)
		return payload, headers, false
	}

	if result.Skipped {
		return nil, nil, true
	}

	newPayload, err := json.Marshal(result.Payload)
	if err != nil {
		slog.Error("action transform: failed to marshal result payload, using original", "error", err, "action_id", action.ID)
		return payload, headers, false
	}

	newHeaders, err := json.Marshal(result.Headers)
	if err != nil {
		slog.Error("action transform: failed to marshal result headers, using original", "error", err, "action_id", action.ID)
		return payload, headers, false
	}

	return newPayload, newHeaders, false
}

func filterActions(all []model.Action, kept []script.ActionRef) []model.Action {
	keptIDs := make(map[uuid.UUID]bool, len(kept))
	for _, a := range kept {
		keptIDs[a.ID] = true
	}

	var filtered []model.Action
	for _, a := range all {
		if keptIDs[a.ID] {
			filtered = append(filtered, a)
		}
	}
	return filtered
}

func (w *FanoutWorker) nextRetryTime(attemptNumber int) *time.Time {
	if attemptNumber >= w.maxRetries {
		return nil
	}
	delay := w.retryBaseDelay * time.Duration(math.Pow(2, float64(attemptNumber-1)))
	if delay > 5*time.Minute {
		delay = 5 * time.Minute
	}
	jitter := time.Duration(float64(delay) * (0.75 + rand.Float64()*0.5))
	t := time.Now().Add(jitter)
	return &t
}

func (w *FanoutWorker) pollPending(ctx context.Context) {
	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			deliveries, err := w.store.Deliveries.ListPending(ctx, 100)
			if err != nil {
				slog.Error("poll pending error", "error", err)
				continue
			}
			metrics.PendingDeliveries.Set(float64(len(deliveries)))
			for _, d := range deliveries {
				slog.Info("catch-up: processing pending delivery", "delivery_id", d.ID)
				w.processDelivery(ctx, d.ID, false)
			}
		}
	}
}

func (w *FanoutWorker) pollRetries(ctx context.Context) {
	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			attempts, err := w.store.Deliveries.ListRetryableAttempts(ctx, 100)
			if err != nil {
				slog.Error("poll retries error", "error", err)
				continue
			}
			metrics.RetryableAttempts.Set(float64(len(attempts)))
			for _, a := range attempts {
				w.retryAttempt(ctx, &a)
			}
		}
	}
}

func (w *FanoutWorker) retryAttempt(ctx context.Context, prev *model.DeliveryAttempt) {
	delivery, err := w.store.Deliveries.GetByID(ctx, prev.DeliveryID)
	if err != nil {
		slog.Error("retry: failed to get delivery", "error", err)
		return
	}

	action, err := w.store.Actions.GetByID(ctx, prev.ActionID)
	if err != nil {
		slog.Error("retry: failed to get action", "error", err)
		return
	}

	nextAttempt := prev.AttemptNumber + 1
	w.dispatchToAction(ctx, delivery, action, nextAttempt)

	// Clear retry schedule on the previous attempt now that we've retried it
	w.store.Deliveries.UpdateAttempt(ctx, prev.ID, model.AttemptFailed, prev.ResponseStatus, prev.ResponseBody, prev.ErrorMessage, nil)

	w.rollUpDeliveryStatus(ctx, delivery.ID)
}

func (w *FanoutWorker) rollUpDeliveryStatus(ctx context.Context, deliveryID uuid.UUID) {
	delivery, err := w.store.Deliveries.GetByID(ctx, deliveryID)
	if err != nil {
		return
	}

	actions, err := w.store.Actions.ListActiveBySource(ctx, delivery.SourceID)
	if err != nil {
		return
	}

	attempts, err := w.store.Deliveries.ListAttemptsByDelivery(ctx, deliveryID)
	if err != nil {
		return
	}

	// Build map of actionID → latest attempt (highest attempt number)
	latest := make(map[uuid.UUID]*model.DeliveryAttempt, len(actions))
	for i := range attempts {
		a := &attempts[i]
		if prev, ok := latest[a.ActionID]; !ok || a.AttemptNumber > prev.AttemptNumber {
			latest[a.ActionID] = a
		}
	}

	anyFailed := false
	for _, action := range actions {
		att, ok := latest[action.ID]
		if !ok {
			// No attempt yet for this action — still in progress
			return
		}
		switch att.Status {
		case model.AttemptSuccess:
			continue
		case model.AttemptFailed:
			if att.AttemptNumber >= w.maxRetries {
				anyFailed = true
			} else {
				// Still has retries remaining — not terminal
				return
			}
		default:
			// Pending or unknown — not terminal
			return
		}
	}

	if anyFailed {
		w.store.Deliveries.UpdateStatus(ctx, deliveryID, model.DeliveryFailed)
	} else {
		w.store.Deliveries.UpdateStatus(ctx, deliveryID, model.DeliveryCompleted)
	}
}
