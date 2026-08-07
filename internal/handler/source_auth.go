package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/zachbroad/nitrohook/internal/inboundauth"
	"github.com/zachbroad/nitrohook/internal/model"
)

// sourceResponse shadows the raw auth_config on model.Source with a
// sanitized view so API responses never carry secret/token material.
type sourceResponse struct {
	*model.Source
	AuthConfig inboundauth.Sanitized `json:"auth_config"`
}

func sanitizeSource(src *model.Source) sourceResponse {
	return sourceResponse{Source: src, AuthConfig: inboundauth.Sanitize(src.AuthConfig)}
}

func sanitizeSources(srcs []model.Source) []sourceResponse {
	out := make([]sourceResponse, len(srcs))
	for i := range srcs {
		out[i] = sanitizeSource(&srcs[i])
	}
	return out
}

// ListAuthPresets returns the provider presets selectable in the auth UI.
func (h *SourceHandler) ListAuthPresets(c *gin.Context) {
	c.JSON(http.StatusOK, inboundauth.PresetNames())
}

type updateSourceAuthRequest struct {
	Enabled   bool   `json:"enabled"`
	Preset    string `json:"preset,omitempty"`
	Secret    string `json:"secret,omitempty"`
	PublicKey string `json:"public_key,omitempty"`
}

// UpdateAuth replaces a source's incoming-webhook auth config. Disabling
// clears the config; enabling requires a known preset plus whichever
// credential that preset needs.
func (h *SourceHandler) UpdateAuth(c *gin.Context) {
	slug := c.Param("sourceSlug")

	var req updateSourceAuthRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.String(http.StatusBadRequest, "invalid request body")
		return
	}

	var raw json.RawMessage
	if req.Enabled {
		base, ok := inboundauth.Preset(req.Preset)
		if !ok {
			c.String(http.StatusBadRequest, "unknown preset")
			return
		}
		var info inboundauth.PresetInfo
		for _, p := range inboundauth.PresetNames() {
			if p.Name == req.Preset {
				info = p
				break
			}
		}

		secret := strings.TrimSpace(req.Secret)
		publicKey := strings.TrimSpace(req.PublicKey)
		// Refuse to store an enabled config with an empty credential:
		// verification would fail open/closed unpredictably, and the UI never
		// re-sends an existing secret, so blank must not overwrite one.
		if info.NeedsSecret && secret == "" {
			c.String(http.StatusBadRequest, "a secret is required for this preset")
			return
		}
		if info.NeedsPublicKey && publicKey == "" {
			c.String(http.StatusBadRequest, "a public key is required for this preset")
			return
		}

		base.Secret = secret
		base.Token = secret // token/bearer schemes reuse the secret field
		base.PublicKey = publicKey
		encoded, err := json.Marshal(base)
		if err != nil {
			c.String(http.StatusInternalServerError, "failed to encode auth config")
			return
		}
		raw = encoded
	}

	src, err := h.store.Sources.SetAuthConfig(c.Request.Context(), slug, raw)
	if err != nil {
		if strings.Contains(err.Error(), "source not found") {
			c.String(http.StatusNotFound, "source not found")
			return
		}
		c.String(http.StatusInternalServerError, "failed to save auth config")
		return
	}

	c.JSON(http.StatusOK, sanitizeSource(src))
}
