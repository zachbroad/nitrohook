package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

// Validation failures return before any store access, so a nil store is safe.
func authRequest(t *testing.T, body string) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := NewSourceHandler(nil)
	r.PUT("/api/sources/:sourceSlug/auth", h.UpdateAuth)
	req := httptest.NewRequest(http.MethodPut, "/api/sources/test/auth", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestUpdateAuthRejectsUnknownPreset(t *testing.T) {
	w := authRequest(t, `{"enabled":true,"preset":"nope","secret":"s"}`)
	if w.Code != http.StatusBadRequest || !strings.Contains(w.Body.String(), "unknown preset") {
		t.Fatalf("expected 400 unknown preset, got %d %q", w.Code, w.Body.String())
	}
}

func TestUpdateAuthRequiresSecret(t *testing.T) {
	w := authRequest(t, `{"enabled":true,"preset":"github","secret":"   "}`)
	if w.Code != http.StatusBadRequest || !strings.Contains(w.Body.String(), "secret is required") {
		t.Fatalf("expected 400 secret required, got %d %q", w.Code, w.Body.String())
	}
}

func TestUpdateAuthRequiresPublicKey(t *testing.T) {
	w := authRequest(t, `{"enabled":true,"preset":"discord-ed25519"}`)
	if w.Code != http.StatusBadRequest || !strings.Contains(w.Body.String(), "public key is required") {
		t.Fatalf("expected 400 public key required, got %d %q", w.Code, w.Body.String())
	}
}

func TestUpdateAuthRejectsInvalidBody(t *testing.T) {
	w := authRequest(t, `{`)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid body, got %d", w.Code)
	}
}

func TestListAuthPresets(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := NewSourceHandler(nil)
	r.GET("/api/auth/presets", h.ListAuthPresets)
	req := httptest.NewRequest(http.MethodGet, "/api/auth/presets", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	body := w.Body.String()
	for _, want := range []string{`"name":"github"`, `"needs_secret":true`, `"name":"discord-ed25519"`, `"needs_public_key":true`} {
		if !strings.Contains(body, want) {
			t.Fatalf("presets response missing %s: %s", want, body)
		}
	}
}
