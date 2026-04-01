package web

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/zachbroad/nitrohook/internal/model"
	"github.com/zachbroad/nitrohook/internal/script"
	"github.com/zachbroad/nitrohook/internal/store"
)

// buildActionConfig extracts action-type-specific fields from the form and
// returns the marshalled config JSON. Returns an empty (nil) RawMessage for
// action types that don't use config (webhook, javascript).
func buildActionConfig(c *gin.Context, actionType model.ActionType) (json.RawMessage, error) {
	var cfg map[string]any

	switch actionType {
	case model.ActionTypeSlack:
		webhookURL := strings.TrimSpace(c.PostForm("slack_webhook_url"))
		if webhookURL == "" {
			return nil, nil
		}
		cfg = map[string]any{"webhook_url": webhookURL}
		if ch := strings.TrimSpace(c.PostForm("slack_channel")); ch != "" {
			cfg["channel"] = ch
		}
		if un := strings.TrimSpace(c.PostForm("slack_username")); un != "" {
			cfg["username"] = un
		}

	case model.ActionTypeSMTP:
		host := strings.TrimSpace(c.PostForm("smtp_host"))
		from := strings.TrimSpace(c.PostForm("smtp_from"))
		to := strings.TrimSpace(c.PostForm("smtp_to"))
		portStr := strings.TrimSpace(c.PostForm("smtp_port"))
		if host == "" || from == "" || to == "" || portStr == "" {
			return nil, nil
		}
		port, _ := strconv.Atoi(portStr)
		cfg = map[string]any{"host": host, "port": port, "from": from, "to": to}
		if u := strings.TrimSpace(c.PostForm("smtp_username")); u != "" {
			cfg["username"] = u
		}
		if p := strings.TrimSpace(c.PostForm("smtp_password")); p != "" {
			cfg["password"] = p
		}
		if s := strings.TrimSpace(c.PostForm("smtp_subject")); s != "" {
			cfg["subject"] = s
		}

	case model.ActionTypeTwilio:
		accountSID := strings.TrimSpace(c.PostForm("twilio_account_sid"))
		authToken := strings.TrimSpace(c.PostForm("twilio_auth_token"))
		from := strings.TrimSpace(c.PostForm("twilio_from"))
		to := strings.TrimSpace(c.PostForm("twilio_to"))
		if accountSID == "" || authToken == "" || from == "" || to == "" {
			return nil, nil
		}
		cfg = map[string]any{"account_sid": accountSID, "auth_token": authToken, "from": from, "to": to}
		if t := strings.TrimSpace(c.PostForm("twilio_body_template")); t != "" {
			cfg["body_template"] = t
		}
	}

	if cfg == nil {
		return nil, nil
	}
	return json.Marshal(cfg)
}

func (h *Handler) CreateAction(c *gin.Context) {
	slug := c.Param("slug")
	source, err := h.store.Sources.GetBySlug(c.Request.Context(), slug)
	if err != nil {
		c.String(http.StatusNotFound, "Source not found")
		return
	}

	actionType := model.ActionType(c.PostForm("type"))
	if actionType == "" {
		actionType = model.ActionTypeWebhook
	}

	var transformScript *string
	if ts := strings.TrimSpace(c.PostForm("transform_script")); ts != "" {
		if err := script.ValidateActionTransform(ts); err != nil {
			slog.Error("invalid transform script", "error", err)
			actions, err := h.store.Actions.List(c.Request.Context(), source.ID)
			if err != nil {
				slog.Error("failed to list actions", "error", err)
			}
			h.renderFragment(c, "source-actions", "actions-card", sourceData{
				Source:  source,
				Actions: actions,
			})
			return
		}
		transformScript = &ts
	}

	params := store.ActionCreateParams{
		SourceID:        source.ID,
		Type:            actionType,
		TransformScript: transformScript,
	}

	switch actionType {
	case model.ActionTypeWebhook:
		targetURL := strings.TrimSpace(c.PostForm("target_url"))
		if targetURL == "" {
			break
		}
		params.TargetURL = &targetURL
		if s := strings.TrimSpace(c.PostForm("signing_secret")); s != "" {
			params.SigningSecret = &s
		}
	case model.ActionTypeJavascript:
		scriptBody := strings.TrimSpace(c.PostForm("script_body"))
		if scriptBody == "" {
			break
		}
		if err := script.ValidateAction(scriptBody); err != nil {
			slog.Error("invalid action script", "error", err)
			break
		}
		params.ScriptBody = &scriptBody
	default:
		cfgJSON, err := buildActionConfig(c, actionType)
		if err != nil {
			slog.Error("failed to build action config", "error", err)
			break
		}
		params.Config = cfgJSON
	}

	if _, err := h.store.Actions.Create(c.Request.Context(), params); err != nil {
		slog.Error("failed to create action", "error", err)
	}

	actions, err := h.store.Actions.List(c.Request.Context(), source.ID)
	if err != nil {
		slog.Error("failed to list actions", "error", err)
	}
	h.renderFragment(c, "source-actions", "actions-card", sourceData{
		Source:  source,
		Actions: actions,
	})
}

func (h *Handler) EditAction(c *gin.Context) {
	slug := c.Param("slug")
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid action ID")
		return
	}
	source, err := h.store.Sources.GetBySlug(c.Request.Context(), slug)
	if err != nil {
		c.String(http.StatusNotFound, "Source not found")
		return
	}
	action, err := h.store.Actions.GetByID(c.Request.Context(), id)
	if err != nil {
		c.String(http.StatusNotFound, "Action not found")
		return
	}
	actions, err := h.store.Actions.List(c.Request.Context(), source.ID)
	if err != nil {
		slog.Error("failed to list actions", "error", err)
	}
	h.renderFragment(c, "source-actions", "action-edit-card", sourceData{
		Source:     source,
		Actions:    actions,
		EditAction: action,
	})
}

func (h *Handler) UpdateAction(c *gin.Context) {
	slug := c.Param("slug")
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid action ID")
		return
	}
	source, err := h.store.Sources.GetBySlug(c.Request.Context(), slug)
	if err != nil {
		c.String(http.StatusNotFound, "Source not found")
		return
	}
	action, err := h.store.Actions.GetByID(c.Request.Context(), id)
	if err != nil {
		c.String(http.StatusNotFound, "Action not found")
		return
	}

	var transformScript *string
	if ts := strings.TrimSpace(c.PostForm("transform_script")); ts != "" {
		if tsErr := script.ValidateActionTransform(ts); tsErr != nil {
			action, _ = h.store.Actions.GetByID(c.Request.Context(), id)
			actions, err := h.store.Actions.List(c.Request.Context(), source.ID)
			if err != nil {
				slog.Error("failed to list actions", "error", err)
			}
			h.renderFragment(c, "source-actions", "action-edit-card", sourceData{
				Source:      source,
				Actions:     actions,
				EditAction:  action,
				ActionError: "Invalid transform script: " + tsErr.Error(),
			})
			return
		}
		transformScript = &ts
	}

	var actionError string
	params := store.ActionUpdateParams{TransformScript: transformScript}

	switch action.Type {
	case model.ActionTypeWebhook:
		targetURL := strings.TrimSpace(c.PostForm("target_url"))
		if targetURL == "" {
			actionError = "Target URL is required for webhook actions"
		} else {
			params.TargetURL = &targetURL
			if s := strings.TrimSpace(c.PostForm("signing_secret")); s != "" {
				params.SigningSecret = &s
			}
		}
	case model.ActionTypeJavascript:
		scriptBody := strings.TrimSpace(c.PostForm("script_body"))
		if scriptBody == "" {
			actionError = "Script body is required for javascript actions"
		} else if err := script.ValidateAction(scriptBody); err != nil {
			actionError = "Invalid script: " + err.Error()
		} else {
			params.ScriptBody = &scriptBody
		}
	default:
		cfgJSON, err := buildActionConfig(c, action.Type)
		if err != nil {
			actionError = "Failed to build action config"
		} else {
			params.Config = cfgJSON
		}
	}

	if actionError == "" {
		if _, err := h.store.Actions.Update(c.Request.Context(), id, params); err != nil {
			slog.Error("failed to update action", "error", err)
			actionError = "Failed to update action"
		}
	}

	if actionError != "" {
		action, _ = h.store.Actions.GetByID(c.Request.Context(), id)
		actions, err := h.store.Actions.List(c.Request.Context(), source.ID)
		if err != nil {
			slog.Error("failed to list actions", "error", err)
		}
		h.renderFragment(c, "source-actions", "action-edit-card", sourceData{
			Source:      source,
			Actions:     actions,
			EditAction:  action,
			ActionError: actionError,
		})
		return
	}

	actions, err := h.store.Actions.List(c.Request.Context(), source.ID)
	if err != nil {
		slog.Error("failed to list actions", "error", err)
	}
	h.renderFragment(c, "source-actions", "actions-card", sourceData{
		Source:        source,
		Actions:       actions,
		ActionSuccess: "Action updated",
	})
}

func (h *Handler) ToggleAction(c *gin.Context) {
	slug := c.Param("slug")
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid action ID")
		return
	}
	source, err := h.store.Sources.GetBySlug(c.Request.Context(), slug)
	if err != nil {
		c.String(http.StatusNotFound, "Source not found")
		return
	}
	isActive := c.PostForm("is_active") == "on"
	if _, err := h.store.Actions.Update(c.Request.Context(), id, store.ActionUpdateParams{IsActive: &isActive}); err != nil {
		slog.Error("failed to toggle action", "error", err)
	}
	actions, err := h.store.Actions.List(c.Request.Context(), source.ID)
	if err != nil {
		slog.Error("failed to list actions", "error", err)
	}
	h.renderFragment(c, "source-actions", "actions-card", sourceData{
		Source:  source,
		Actions: actions,
	})
}

func (h *Handler) DeleteAction(c *gin.Context) {
	slug := c.Param("slug")
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.String(http.StatusBadRequest, "Invalid action ID")
		return
	}
	source, err := h.store.Sources.GetBySlug(c.Request.Context(), slug)
	if err != nil {
		c.String(http.StatusNotFound, "Source not found")
		return
	}
	if err := h.store.Actions.Delete(c.Request.Context(), id); err != nil {
		slog.Error("failed to delete action", "error", err)
	}
	actions, err := h.store.Actions.List(c.Request.Context(), source.ID)
	if err != nil {
		slog.Error("failed to list actions", "error", err)
	}
	h.renderFragment(c, "source-actions", "actions-card", sourceData{
		Source:  source,
		Actions: actions,
	})
}
