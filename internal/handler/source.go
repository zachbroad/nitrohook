package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/zachbroad/nitrohook/internal/script"
	"github.com/zachbroad/nitrohook/internal/store"
)

type SourceHandler struct {
	store *store.Store
}

func NewSourceHandler(s *store.Store) *SourceHandler {
	return &SourceHandler{store: s}
}

type createSourceRequest struct {
	Name       string  `json:"name"`
	Slug       string  `json:"slug,omitempty"`
	Mode       string  `json:"mode,omitempty"`
	ScriptBody *string `json:"script_body,omitempty"`
}

type updateSourceRequest struct {
	Name       *string `json:"name,omitempty"`
	Mode       *string `json:"mode,omitempty"`
	ScriptBody *string `json:"script_body,omitempty"`
}

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

func validateMode(mode string) bool {
	return mode == "record" || mode == "active"
}

func (h *SourceHandler) List(c *gin.Context) {
	sources, err := h.store.Sources.List(c.Request.Context())
	if err != nil {
		slog.Error("failed to list sources", "error", err)
		c.String(http.StatusInternalServerError, "failed to list sources")
		return
	}

	if sources == nil {
		c.Data(http.StatusOK, "application/json", []byte("[]"))
		return
	}
	c.JSON(http.StatusOK, sources)
}

func (h *SourceHandler) Create(c *gin.Context) {
	var req createSourceRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.String(http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Name == "" {
		c.String(http.StatusBadRequest, "name is required")
		return
	}

	slug := req.Slug
	if slug == "" {
		slug = generateSlug(req.Name)
	}
	if slug == "" {
		c.String(http.StatusBadRequest, "could not generate slug from name")
		return
	}

	// Default to record mode for new sources
	mode := req.Mode
	if mode == "" {
		mode = "record"
	}
	if !validateMode(mode) {
		c.String(http.StatusBadRequest, "mode must be 'record' or 'active'")
		return
	}

	// Validate script if provided
	if req.ScriptBody != nil && *req.ScriptBody != "" {
		if err := script.Validate(*req.ScriptBody); err != nil {
			c.String(http.StatusBadRequest, "invalid script: "+err.Error())
			return
		}
	}

	src, err := h.store.Sources.Create(c.Request.Context(), req.Name, slug, mode, req.ScriptBody)
	if err != nil {
		if strings.Contains(err.Error(), "duplicate key") || strings.Contains(err.Error(), "unique") {
			c.String(http.StatusConflict, "source with this slug already exists")
			return
		}
		c.String(http.StatusInternalServerError, "failed to create source")
		return
	}

	c.JSON(http.StatusCreated, src)
}

func (h *SourceHandler) Get(c *gin.Context) {
	slug := c.Param("sourceSlug")

	src, err := h.store.Sources.GetBySlug(c.Request.Context(), slug)
	if err != nil {
		c.String(http.StatusNotFound, "source not found")
		return
	}

	c.JSON(http.StatusOK, src)
}

func (h *SourceHandler) Update(c *gin.Context) {
	slug := c.Param("sourceSlug")

	var req updateSourceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.String(http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Mode != nil && !validateMode(*req.Mode) {
		c.String(http.StatusBadRequest, "mode must be 'record' or 'active'")
		return
	}

	// Validate script if provided and non-empty
	if req.ScriptBody != nil && *req.ScriptBody != "" {
		if err := script.Validate(*req.ScriptBody); err != nil {
			c.String(http.StatusBadRequest, "invalid script: "+err.Error())
			return
		}
	}

	// Empty string means "clear the script"
	clearScript := req.ScriptBody != nil && *req.ScriptBody == ""

	src, err := h.store.Sources.Update(c.Request.Context(), slug, req.Name, req.Mode, req.ScriptBody, clearScript)
	if err != nil {
		if strings.Contains(err.Error(), "source not found") {
			c.String(http.StatusNotFound, "source not found")
			return
		}
		c.String(http.StatusInternalServerError, "failed to update source")
		return
	}

	c.JSON(http.StatusOK, src)
}

type testScriptRequest struct {
	ScriptBody string `json:"script_body"`
	DeliveryID string `json:"delivery_id"`
}

// TestScript runs a candidate source-transform script against a recorded
// delivery and returns the transform result as JSON. It never returns 500 for
// a script that merely throws — that surfaces as {"result":null,"error":...}.
func (h *SourceHandler) TestScript(c *gin.Context) {
	slug := c.Param("sourceSlug")
	var req testScriptRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.String(http.StatusBadRequest, "invalid request body")
		return
	}
	if strings.TrimSpace(req.DeliveryID) == "" {
		c.String(http.StatusBadRequest, "delivery_id is required")
		return
	}
	if strings.TrimSpace(req.ScriptBody) == "" {
		c.String(http.StatusBadRequest, "script_body is required")
		return
	}

	source, err := h.store.Sources.GetBySlug(c.Request.Context(), slug)
	if err != nil {
		c.String(http.StatusNotFound, "source not found")
		return
	}
	did, err := uuid.Parse(req.DeliveryID)
	if err != nil {
		c.String(http.StatusBadRequest, "invalid delivery_id")
		return
	}
	delivery, err := h.store.Deliveries.GetByID(c.Request.Context(), did)
	if err != nil || delivery.SourceID != source.ID {
		c.String(http.StatusNotFound, "delivery not found for this source")
		return
	}

	// Mirror worker.FanoutWorker.runTransform's unmarshal behavior exactly so
	// a passing test here implies the script will run the same way in
	// production: any unmarshal failure is a handled error, not a fallback.
	var payload map[string]any
	if err := json.Unmarshal(delivery.Payload, &payload); err != nil {
		c.JSON(http.StatusOK, gin.H{"result": nil, "error": "unmarshal payload: " + err.Error()})
		return
	}
	var headers map[string]string
	if err := json.Unmarshal(delivery.Headers, &headers); err != nil {
		c.JSON(http.StatusOK, gin.H{"result": nil, "error": "unmarshal headers: " + err.Error()})
		return
	}

	actions, err := h.store.Actions.ListActiveBySource(c.Request.Context(), source.ID)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"result": nil, "error": "failed to load actions: " + err.Error()})
		return
	}
	actionRefs := make([]script.ActionRef, len(actions))
	for i, a := range actions {
		targetURL := ""
		if a.TargetURL != nil {
			targetURL = *a.TargetURL
		}
		actionRefs[i] = script.ActionRef{ID: a.ID, TargetURL: targetURL}
	}

	result, err := script.Run(req.ScriptBody, script.TransformInput{
		Payload: payload, Headers: headers, Actions: actionRefs,
	})
	if err != nil {
		msg := err.Error()
		c.JSON(http.StatusOK, gin.H{"result": nil, "error": msg})
		return
	}
	c.JSON(http.StatusOK, gin.H{"result": result, "error": nil})
}

func (h *SourceHandler) Delete(c *gin.Context) {
	slug := c.Param("sourceSlug")

	if err := h.store.Sources.Delete(c.Request.Context(), slug); err != nil {
		if strings.Contains(err.Error(), "source not found") {
			c.String(http.StatusNotFound, "source not found")
			return
		}
		c.String(http.StatusInternalServerError, "failed to delete source")
		return
	}

	c.Status(http.StatusNoContent)
}
