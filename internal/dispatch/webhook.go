package dispatch

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/zachbroad/nitrohook/internal/model"
	"github.com/zachbroad/nitrohook/internal/signing"
)

const maxBodyLen = 4096

type WebhookDispatcher struct {
	Client *http.Client
}

func (d *WebhookDispatcher) Validate(action *model.Action) error {
	if action.TargetURL == nil || *action.TargetURL == "" {
		return fmt.Errorf("target_url is required for webhook actions")
	}
	return nil
}

func (d *WebhookDispatcher) Dispatch(ctx context.Context, action *model.Action, deliveryID string, payload, headers json.RawMessage) Result {
	targetURL := ""
	if action.TargetURL != nil {
		targetURL = *action.TargetURL
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, targetURL, bytes.NewReader(payload))
	if err != nil {
		errMsg := err.Error()
		return Result{ErrorMessage: &errMsg}
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Delivery-ID", deliveryID)

	var headerMap map[string]string
	if err := json.Unmarshal(headers, &headerMap); err == nil {
		for k, v := range headerMap {
			if k != "Content-Type" {
				req.Header.Set(k, v)
			}
		}
	}

	if action.SigningSecret != nil {
		sig := signing.Sign(payload, *action.SigningSecret)
		req.Header.Set("X-Webhook-Signature-256", sig)
	}

	resp, err := d.Client.Do(req)
	if err != nil {
		errMsg := err.Error()
		return Result{ErrorMessage: &errMsg}
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, maxBodyLen))
	bodyStr := string(body)
	statusCode := resp.StatusCode

	if statusCode >= 200 && statusCode < 300 {
		return Result{Success: true, ResponseStatus: &statusCode, ResponseBody: &bodyStr}
	}

	errMsg := fmt.Sprintf("HTTP %d", statusCode)
	return Result{ResponseStatus: &statusCode, ResponseBody: &bodyStr, ErrorMessage: &errMsg}
}
