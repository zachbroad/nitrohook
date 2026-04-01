package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/zachbroad/nitrohook/internal/dispatch"
	"github.com/zachbroad/nitrohook/internal/model"
	"github.com/zachbroad/nitrohook/internal/script"
	"github.com/zachbroad/nitrohook/internal/store"
)

type ActionHandler struct {
	store *store.Store
}

func NewActionHandler(s *store.Store) *ActionHandler {
	return &ActionHandler{store: s}
}

type createActionRequest struct {
	Type            string  `json:"type"`
	TargetURL       *string `json:"target_url,omitempty"`
	SigningSecret   *string `json:"signing_secret,omitempty"`
	ScriptBody      *string `json:"script_body,omitempty"`
	TransformScript *string `json:"transform_script,omitempty"`
}

type updateActionRequest struct {
	TargetURL       *string `json:"target_url,omitempty"`
	SigningSecret   *string `json:"signing_secret,omitempty"`
	IsActive        *bool   `json:"is_active,omitempty"`
	TransformScript *string `json:"transform_script,omitempty"`
}

func (h *ActionHandler) Create(c *gin.Context) {
	sourceSlug := c.Param("sourceSlug")

	src, err := h.store.Sources.GetBySlug(c.Request.Context(), sourceSlug)
	if err != nil {
		c.String(http.StatusNotFound, "source not found")
		return
	}

	var req createActionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.String(http.StatusBadRequest, "invalid request body")
		return
	}

	actionType := model.ActionType(req.Type)
	if actionType == "" {
		actionType = model.ActionTypeWebhook
	}

	d, err := dispatch.Get(actionType)
	if err != nil {
		c.String(http.StatusBadRequest, "invalid action type: %s", req.Type)
		return
	}

	// Build a temporary action to validate
	tmpAction := &model.Action{
		Type:          actionType,
		TargetURL:     req.TargetURL,
		SigningSecret: req.SigningSecret,
		ScriptBody:    req.ScriptBody,
	}
	if err := d.Validate(tmpAction); err != nil {
		c.String(http.StatusBadRequest, err.Error())
		return
	}

	if req.TransformScript != nil && *req.TransformScript != "" {
		if err := script.ValidateActionTransform(*req.TransformScript); err != nil {
			c.String(http.StatusBadRequest, "invalid transform script: %s", err.Error())
			return
		}
	}

	action, err := h.store.Actions.Create(c.Request.Context(), store.ActionCreateParams{
		SourceID:        src.ID,
		Type:            actionType,
		TargetURL:       req.TargetURL,
		SigningSecret:   req.SigningSecret,
		ScriptBody:      req.ScriptBody,
		TransformScript: req.TransformScript,
	})
	if err != nil {
		c.String(http.StatusInternalServerError, "failed to create action")
		return
	}

	c.JSON(http.StatusCreated, action)
}

func (h *ActionHandler) List(c *gin.Context) {
	sourceSlug := c.Param("sourceSlug")

	src, err := h.store.Sources.GetBySlug(c.Request.Context(), sourceSlug)
	if err != nil {
		c.String(http.StatusNotFound, "source not found")
		return
	}

	actions, err := h.store.Actions.List(c.Request.Context(), src.ID)
	if err != nil {
		c.String(http.StatusInternalServerError, "failed to list actions")
		return
	}
	c.JSON(http.StatusOK, actions)
}

func (h *ActionHandler) Get(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.String(http.StatusBadRequest, "invalid action id")
		return
	}

	action, err := h.store.Actions.GetByID(c.Request.Context(), id)
	if err != nil {
		c.String(http.StatusNotFound, "action not found")
		return
	}

	c.JSON(http.StatusOK, action)
}

func (h *ActionHandler) Update(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.String(http.StatusBadRequest, "invalid action id")
		return
	}

	var req updateActionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.String(http.StatusBadRequest, "invalid request body")
		return
	}

	if req.TransformScript != nil && *req.TransformScript != "" {
		if err := script.ValidateActionTransform(*req.TransformScript); err != nil {
			c.String(http.StatusBadRequest, "invalid transform script: %s", err.Error())
			return
		}
	}

	action, err := h.store.Actions.Update(c.Request.Context(), id, store.ActionUpdateParams{
		TargetURL:       req.TargetURL,
		SigningSecret:   req.SigningSecret,
		IsActive:        req.IsActive,
		TransformScript: req.TransformScript,
	})
	if err != nil {
		c.String(http.StatusInternalServerError, "failed to update action")
		return
	}

	c.JSON(http.StatusOK, action)
}

func (h *ActionHandler) Delete(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.String(http.StatusBadRequest, "invalid action id")
		return
	}

	if err := h.store.Actions.Delete(c.Request.Context(), id); err != nil {
		c.String(http.StatusInternalServerError, "failed to delete action")
		return
	}

	c.Status(http.StatusNoContent)
}
