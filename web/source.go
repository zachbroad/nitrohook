package web

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/zachbroad/nitrohook/internal/model"
	"github.com/zachbroad/nitrohook/internal/script"
)

var nonAlphanumDash = regexp.MustCompile(`[^a-z0-9-]+`)
var multiDash = regexp.MustCompile(`-{2,}`)

/** Generate a slug based off the input name provided by the user. */
func generateSlug(name string) string {
	s := strings.ToLower(name)
	s = strings.ReplaceAll(s, " ", "-")
	s = nonAlphanumDash.ReplaceAllString(s, "")
	s = multiDash.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	return s
}

func webhookURL(c *gin.Context, slug string) string {
	scheme := "http"
	if c.Request.TLS != nil || c.GetHeader("X-Forwarded-Proto") == "https" {
		scheme = "https"
	}
	return fmt.Sprintf("%s://%s/webhooks/%s", scheme, c.Request.Host, slug)
}

func (h *Handler) Sources(c *gin.Context) {
	sources, err := h.store.Sources.List(c.Request.Context())
	if err != nil {
		slog.Error("failed to list sources", "error", err)
		c.String(http.StatusInternalServerError, "Internal server error")
		return
	}
	counts := make(map[uuid.UUID]int, len(sources))
	for _, s := range sources {
		n, _ := h.store.Actions.CountBySource(c.Request.Context(), s.ID)
		counts[s.ID] = n
	}
	h.render(c, "sources", sourcesData{
		Nav:          "sources",
		Sources:      sources,
		ActionCounts: counts,
	})
}

func (h *Handler) SourceDetail(c *gin.Context) {
	slug := c.Param("slug")
	source, err := h.store.Sources.GetBySlug(c.Request.Context(), slug)
	if err != nil {
		c.String(http.StatusNotFound, "Source not found")
		return
	}
	h.render(c, "source-overview", sourceData{
		Nav:        "sources",
		Source:     source,
		WebhookURL: webhookURL(c, source.Slug),
	})
}

func (h *Handler) SourceScript(c *gin.Context) {
	slug := c.Param("slug")
	source, err := h.store.Sources.GetBySlug(c.Request.Context(), slug)
	if err != nil {
		c.String(http.StatusNotFound, "Source not found")
		return
	}
	deliveries, _ := h.store.Deliveries.List(c.Request.Context(), &slug, 10)
	h.render(c, "source-script", sourceData{
		Nav:        "sources",
		Source:     source,
		Deliveries: deliveries,
	})
}

func (h *Handler) SourceActions(c *gin.Context) {
	slug := c.Param("slug")
	source, err := h.store.Sources.GetBySlug(c.Request.Context(), slug)
	if err != nil {
		c.String(http.StatusNotFound, "Source not found")
		return
	}
	actions, err := h.store.Actions.List(c.Request.Context(), source.ID)
	if err != nil {
		slog.Error("failed to list actions", "error", err)
		c.String(http.StatusInternalServerError, "Internal server error")
		return
	}
	data := sourceData{
		Nav:     "sources",
		Source:  source,
		Actions: actions,
	}
	if c.GetHeader("HX-Request") != "" && c.GetHeader("HX-Boosted") == "" {
		h.renderFragment(c, "source-actions", "actions-card", data)
	} else {
		h.render(c, "source-actions", data)
	}
}

func (h *Handler) SourceEvents(c *gin.Context) {
	slug := c.Param("slug")
	source, err := h.store.Sources.GetBySlug(c.Request.Context(), slug)
	if err != nil {
		c.String(http.StatusNotFound, "Source not found")
		return
	}
	deliveries, _ := h.store.Deliveries.List(c.Request.Context(), &slug, 50)
	h.render(c, "source-events", sourceData{
		Nav:        "sources",
		Source:     source,
		Deliveries: deliveries,
		WebhookURL: webhookURL(c, source.Slug),
	})
}

func (h *Handler) CreateSource(c *gin.Context) {
	name := strings.TrimSpace(c.PostForm("name"))
	if name == "" {
		sources, _ := h.store.Sources.List(c.Request.Context())
		h.render(c, "sources", sourcesData{
			Nav:     "sources",
			Sources: sources,
			Error:   "Name is required",
		})
		return
	}
	slug := generateSlug(name)
	if slug == "" {
		sources, _ := h.store.Sources.List(c.Request.Context())
		h.render(c, "sources", sourcesData{
			Nav:     "sources",
			Sources: sources,
			Error:   "Could not generate slug from name",
		})
		return
	}
	_, err := h.store.Sources.Create(c.Request.Context(), name, slug, "record", nil)
	if err != nil {
		sources, _ := h.store.Sources.List(c.Request.Context())
		errMsg := "Failed to create source"
		if strings.Contains(err.Error(), "duplicate") || strings.Contains(err.Error(), "unique") {
			errMsg = "Source with this slug already exists"
		}
		h.render(c, "sources", sourcesData{
			Nav:     "sources",
			Sources: sources,
			Error:   errMsg,
		})
		return
	}
	c.Redirect(http.StatusSeeOther, "/sources/"+slug)
}

func (h *Handler) UpdateSource(c *gin.Context) {
	slug := c.Param("slug")
	name := strings.TrimSpace(c.PostForm("name"))
	if name != "" {
		if _, err := h.store.Sources.Update(c.Request.Context(), slug, &name, nil, nil, false); err != nil {
			slog.Error("failed to update source", "error", err)
		}
	}
	c.Redirect(http.StatusSeeOther, "/sources/"+slug)
}

func (h *Handler) DeleteSource(c *gin.Context) {
	slug := c.Param("slug")
	if err := h.store.Sources.Delete(c.Request.Context(), slug); err != nil {
		slog.Error("failed to delete source", "error", err)
		c.String(http.StatusInternalServerError, "Failed to delete source")
		return
	}
	c.Header("HX-Redirect", "/sources")
	c.Status(http.StatusOK)
}

func (h *Handler) UpdateSourceMode(c *gin.Context) {
	slug := c.Param("slug")
	mode := c.PostForm("mode")
	if mode != "record" && mode != "active" {
		c.String(http.StatusBadRequest, "Invalid mode")
		return
	}
	source, err := h.store.Sources.Update(c.Request.Context(), slug, nil, &mode, nil, false)
	if err != nil {
		slog.Error("failed to update source mode", "error", err)
		c.String(http.StatusInternalServerError, "Failed to update mode")
		return
	}
	actions, _ := h.store.Actions.List(c.Request.Context(), source.ID)
	h.renderFragment(c, "source-overview", "mode-card", sourceData{
		Source:  source,
		Actions: actions,
	})
}

func (h *Handler) UpdateSourceScript(c *gin.Context) {
	slug := c.Param("slug")
	scriptBody := c.PostForm("script_body")

	source, err := h.store.Sources.GetBySlug(c.Request.Context(), slug)
	if err != nil {
		c.String(http.StatusNotFound, "Source not found")
		return
	}

	var scriptError, scriptSuccess string
	if strings.TrimSpace(scriptBody) == "" {
		source, err = h.store.Sources.Update(c.Request.Context(), slug, nil, nil, nil, true)
		if err != nil {
			slog.Error("failed to clear script", "error", err)
			scriptError = "Failed to clear script"
		} else {
			scriptSuccess = "Script cleared"
		}
	} else {
		if err := script.Validate(scriptBody); err != nil {
			scriptError = "Invalid script: " + err.Error()
		} else {
			source, err = h.store.Sources.Update(c.Request.Context(), slug, nil, nil, &scriptBody, false)
			if err != nil {
				slog.Error("failed to save script", "error", err)
				scriptError = "Failed to save script"
			} else {
				scriptSuccess = "Script saved"
			}
		}
	}

	actions, _ := h.store.Actions.List(c.Request.Context(), source.ID)
	deliveries, _ := h.store.Deliveries.List(c.Request.Context(), &slug, 10)
	h.renderFragment(c, "source-script", "script-card", sourceData{
		Source:        source,
		Actions:       actions,
		Deliveries:    deliveries,
		ScriptError:   scriptError,
		ScriptSuccess: scriptSuccess,
	})
}

func (h *Handler) ClearSourceScript(c *gin.Context) {
	slug := c.Param("slug")
	source, err := h.store.Sources.Update(c.Request.Context(), slug, nil, nil, nil, true)
	if err != nil {
		slog.Error("failed to clear script", "error", err)
		c.String(http.StatusInternalServerError, "Failed to clear script")
		return
	}
	actions, _ := h.store.Actions.List(c.Request.Context(), source.ID)
	deliveries, _ := h.store.Deliveries.List(c.Request.Context(), &slug, 10)
	h.renderFragment(c, "source-script", "script-card", sourceData{
		Source:        source,
		Actions:       actions,
		Deliveries:    deliveries,
		ScriptSuccess: "Script cleared",
	})
}

func (h *Handler) forwardDeliveryToStream(ctx context.Context, deliveryID uuid.UUID) error {
	return h.rdb.XAdd(ctx, &redis.XAddArgs{
		Stream: "deliveries",
		MaxLen: 10000,
		Approx: true,
		Values: map[string]any{
			"delivery_id": deliveryID.String(),
			"force":       "1",
		},
	}).Err()
}

func (h *Handler) ForwardDelivery(c *gin.Context) {
	slug := c.Param("slug")
	idStr := c.Param("id")

	source, err := h.store.Sources.GetBySlug(c.Request.Context(), slug)
	if err != nil {
		c.String(http.StatusNotFound, "Source not found")
		return
	}

	deliveryID, err := uuid.Parse(idStr)
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid delivery ID")
		return
	}

	delivery, err := h.store.Deliveries.GetByID(c.Request.Context(), deliveryID)
	if err != nil || delivery.SourceID != source.ID {
		c.String(http.StatusNotFound, "Delivery not found")
		return
	}

	if delivery.Status != model.DeliveryRecorded {
		c.Header("HX-Refresh", "true")
		c.Status(http.StatusOK)
		return
	}

	if err := h.store.Deliveries.UpdateStatus(c.Request.Context(), deliveryID, model.DeliveryPending); err != nil {
		slog.Error("failed to update delivery status for forward", "error", err, "delivery_id", deliveryID)
		c.String(http.StatusInternalServerError, "Failed to forward delivery")
		return
	}

	if err := h.forwardDeliveryToStream(c.Request.Context(), deliveryID); err != nil {
		slog.Error("failed to publish forced delivery to stream", "error", err, "delivery_id", deliveryID)
		_ = h.store.Deliveries.UpdateStatus(c.Request.Context(), deliveryID, model.DeliveryRecorded)
		c.String(http.StatusInternalServerError, "Failed to forward delivery")
		return
	}

	c.Header("HX-Refresh", "true")
	c.Status(http.StatusOK)
}

func (h *Handler) ForwardAllRecorded(c *gin.Context) {
	slug := c.Param("slug")

	if _, err := h.store.Sources.GetBySlug(c.Request.Context(), slug); err != nil {
		c.String(http.StatusNotFound, "Source not found")
		return
	}

	deliveries, err := h.store.Deliveries.List(c.Request.Context(), &slug, 200)
	if err != nil {
		slog.Error("failed to list deliveries for forward-all", "error", err, "slug", slug)
		c.String(http.StatusInternalServerError, "Failed to list deliveries")
		return
	}

	for _, d := range deliveries {
		if d.Status != model.DeliveryRecorded {
			continue
		}
		if err := h.store.Deliveries.UpdateStatus(c.Request.Context(), d.ID, model.DeliveryPending); err != nil {
			slog.Error("failed to update delivery status for forward-all", "error", err, "delivery_id", d.ID)
			continue
		}
		if err := h.forwardDeliveryToStream(c.Request.Context(), d.ID); err != nil {
			slog.Error("failed to publish delivery to stream for forward-all", "error", err, "delivery_id", d.ID)
			_ = h.store.Deliveries.UpdateStatus(c.Request.Context(), d.ID, model.DeliveryRecorded)
		}
	}

	c.Header("HX-Refresh", "true")
	c.Status(http.StatusOK)
}

func (h *Handler) TestSourceScript(c *gin.Context) {
	slug := c.Param("slug")
	scriptBody := c.PostForm("script_body")
	deliveryID := c.PostForm("delivery_id")

	if strings.TrimSpace(deliveryID) == "" {
		h.renderFragment(c, "source-script", "script-test-result", scriptTestData{
			Error: "Select a payload to test against",
		})
		return
	}

	source, err := h.store.Sources.GetBySlug(c.Request.Context(), slug)
	if err != nil {
		h.renderFragment(c, "source-script", "script-test-result", scriptTestData{
			Error: "Source not found",
		})
		return
	}

	did, err := uuid.Parse(deliveryID)
	if err != nil {
		h.renderFragment(c, "source-script", "script-test-result", scriptTestData{
			Error: "Invalid delivery ID",
		})
		return
	}

	delivery, err := h.store.Deliveries.GetByID(c.Request.Context(), did)
	if err != nil || delivery.SourceID != source.ID {
		h.renderFragment(c, "source-script", "script-test-result", scriptTestData{
			Error: "Delivery not found for this source",
		})
		return
	}

	if strings.TrimSpace(scriptBody) == "" {
		h.renderFragment(c, "source-script", "script-test-result", scriptTestData{
			Error: "Script body is empty",
		})
		return
	}

	var payload map[string]any
	if delivery.Payload != nil {
		if err := json.Unmarshal(delivery.Payload, &payload); err != nil {
			payload = map[string]any{"_raw": string(delivery.Payload)}
		}
	}
	var headers map[string]string
	if delivery.Headers != nil {
		if err := json.Unmarshal(delivery.Headers, &headers); err != nil {
			headers = map[string]string{}
		}
	}

	actions, _ := h.store.Actions.ListActiveBySource(c.Request.Context(), source.ID)
	actionRefs := make([]script.ActionRef, len(actions))
	for i, a := range actions {
		targetURL := ""
		if a.TargetURL != nil {
			targetURL = *a.TargetURL
		}
		actionRefs[i] = script.ActionRef{ID: a.ID, TargetURL: targetURL}
	}

	input := script.TransformInput{
		Payload: payload,
		Headers: headers,
		Actions: actionRefs,
	}

	result, err := script.Run(scriptBody, input)
	if err != nil {
		h.renderFragment(c, "source-script", "script-test-result", scriptTestData{
			Error: err.Error(),
		})
		return
	}

	h.renderFragment(c, "source-script", "script-test-result", scriptTestData{
		Result: result,
	})
}
