package web

import (
	"log/slog"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/zachbroad/nitrohook/internal/model"
	"github.com/zachbroad/nitrohook/internal/script"
)

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

	switch actionType {
	case model.ActionTypeWebhook:
		targetURL := strings.TrimSpace(c.PostForm("target_url"))
		if targetURL != "" {
			var signingSecret *string
			if s := strings.TrimSpace(c.PostForm("signing_secret")); s != "" {
				signingSecret = &s
			}
			if _, err := h.store.Actions.Create(c.Request.Context(), source.ID, actionType, &targetURL, signingSecret, nil); err != nil {
				slog.Error("failed to create action", "error", err)
			}
		}
	case model.ActionTypeJavascript:
		scriptBody := strings.TrimSpace(c.PostForm("script_body"))
		if scriptBody != "" {
			if err := script.ValidateAction(scriptBody); err != nil {
				slog.Error("invalid action script", "error", err)
			} else {
				if _, err := h.store.Actions.Create(c.Request.Context(), source.ID, actionType, nil, nil, &scriptBody); err != nil {
					slog.Error("failed to create action", "error", err)
				}
			}
		}
	}

	actions, _ := h.store.Actions.List(c.Request.Context(), source.ID)
	h.renderFragment(c, "source", "actions-card", sourceData{
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
	actions, _ := h.store.Actions.List(c.Request.Context(), source.ID)
	h.renderFragment(c, "source", "action-edit-card", sourceData{
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

	var actionError string
	switch action.Type {
	case model.ActionTypeWebhook:
		targetURL := strings.TrimSpace(c.PostForm("target_url"))
		if targetURL == "" {
			actionError = "Target URL is required for webhook actions"
		} else {
			var signingSecret *string
			if s := strings.TrimSpace(c.PostForm("signing_secret")); s != "" {
				signingSecret = &s
			}
			if _, err := h.store.Actions.Update(c.Request.Context(), id, &targetURL, signingSecret, nil, nil); err != nil {
				slog.Error("failed to update action", "error", err)
				actionError = "Failed to update action"
			}
		}
	case model.ActionTypeJavascript:
		scriptBody := strings.TrimSpace(c.PostForm("script_body"))
		if scriptBody == "" {
			actionError = "Script body is required for javascript actions"
		} else if err := script.ValidateAction(scriptBody); err != nil {
			actionError = "Invalid script: " + err.Error()
		} else {
			if _, err := h.store.Actions.Update(c.Request.Context(), id, nil, nil, nil, &scriptBody); err != nil {
				slog.Error("failed to update action", "error", err)
				actionError = "Failed to update action"
			}
		}
	}

	if actionError != "" {
		action, _ = h.store.Actions.GetByID(c.Request.Context(), id)
		actions, _ := h.store.Actions.List(c.Request.Context(), source.ID)
		h.renderFragment(c, "source", "action-edit-card", sourceData{
			Source:      source,
			Actions:     actions,
			EditAction:  action,
			ActionError: actionError,
		})
		return
	}

	actions, _ := h.store.Actions.List(c.Request.Context(), source.ID)
	h.renderFragment(c, "source", "actions-card", sourceData{
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
	if _, err := h.store.Actions.Update(c.Request.Context(), id, nil, nil, &isActive, nil); err != nil {
		slog.Error("failed to toggle action", "error", err)
	}
	actions, _ := h.store.Actions.List(c.Request.Context(), source.ID)
	h.renderFragment(c, "source", "actions-card", sourceData{
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
	actions, _ := h.store.Actions.List(c.Request.Context(), source.ID)
	h.renderFragment(c, "source", "actions-card", sourceData{
		Source:  source,
		Actions: actions,
	})
}
