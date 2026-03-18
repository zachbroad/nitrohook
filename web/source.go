package web

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/zachbroad/nitrohook/internal/script"
)

var nonAlphanumDash = regexp.MustCompile(`[^a-z0-9-]+`)
var multiDash = regexp.MustCompile(`-{2,}`)

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
	h.render(c, "sources", sourcesData{
		Nav:     "sources",
		Sources: sources,
	})
}

func (h *Handler) SourceDetail(c *gin.Context) {
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
	deliveries, _ := h.store.Deliveries.List(c.Request.Context(), &slug, 10)
	h.render(c, "source", sourceData{
		Nav:        "sources",
		Source:     source,
		Actions:    actions,
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
	h.renderFragment(c, "source", "mode-card", sourceData{
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
	h.renderFragment(c, "source", "script-card", sourceData{
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
	h.renderFragment(c, "source", "script-card", sourceData{
		Source:        source,
		Actions:       actions,
		Deliveries:    deliveries,
		ScriptSuccess: "Script cleared",
	})
}

func (h *Handler) TestSourceScript(c *gin.Context) {
	slug := c.Param("slug")
	scriptBody := c.PostForm("script_body")
	deliveryID := c.PostForm("delivery_id")

	if strings.TrimSpace(deliveryID) == "" {
		h.renderFragment(c, "source", "script-test-result", scriptTestData{
			Error: "Select a payload to test against",
		})
		return
	}

	source, err := h.store.Sources.GetBySlug(c.Request.Context(), slug)
	if err != nil {
		h.renderFragment(c, "source", "script-test-result", scriptTestData{
			Error: "Source not found",
		})
		return
	}

	did, err := uuid.Parse(deliveryID)
	if err != nil {
		h.renderFragment(c, "source", "script-test-result", scriptTestData{
			Error: "Invalid delivery ID",
		})
		return
	}

	delivery, err := h.store.Deliveries.GetByID(c.Request.Context(), did)
	if err != nil || delivery.SourceID != source.ID {
		h.renderFragment(c, "source", "script-test-result", scriptTestData{
			Error: "Delivery not found for this source",
		})
		return
	}

	if strings.TrimSpace(scriptBody) == "" {
		h.renderFragment(c, "source", "script-test-result", scriptTestData{
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
		h.renderFragment(c, "source", "script-test-result", scriptTestData{
			Error: err.Error(),
		})
		return
	}

	h.renderFragment(c, "source", "script-test-result", scriptTestData{
		Result: result,
	})
}
