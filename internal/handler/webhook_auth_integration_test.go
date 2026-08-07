//go:build integration

package handler_test

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/zachbroad/nitrohook/internal/store"
)

func countDeliveries(t *testing.T, s *store.Store, slug string) int {
	t.Helper()
	deliveries, err := s.Deliveries.List(context.Background(), &slug, 100)
	if err != nil {
		t.Fatalf("list deliveries: %v", err)
	}
	return len(deliveries)
}

func TestIngestRejectsBadSignature(t *testing.T) {
	registerTestDispatchers()
	r, store, cleanup := setupRouter(t) // refactored to return the store
	defer cleanup()
	createSource(t, r, "Signed Source", "signed-source")

	cfg, _ := json.Marshal(map[string]any{
		"scheme": "hmac", "algo": "sha256", "encoding": "hex",
		"sig_header": "X-Hub-Signature-256", "sig_parser": "plain",
		"sig_prefix": "sha256=", "template": "raw_body", "secret": "topsecret",
	})
	if _, err := store.Sources.SetAuthConfig(context.Background(), "signed-source", cfg); err != nil {
		t.Fatalf("set auth: %v", err)
	}

	body := []byte(`{"event":"push"}`)

	// No signature -> 401, nothing stored.
	req := httptest.NewRequest(http.MethodPost, "/webhooks/signed-source", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 with no signature, got %d", w.Code)
	}
	if n := countDeliveries(t, store, "signed-source"); n != 0 {
		t.Fatalf("expected 0 deliveries after rejection, got %d", n)
	}

	// Valid signature -> 202.
	mac := hmac.New(sha256.New, []byte("topsecret"))
	mac.Write(body)
	req2 := httptest.NewRequest(http.MethodPost, "/webhooks/signed-source", bytes.NewReader(body))
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("X-Hub-Signature-256", "sha256="+hex.EncodeToString(mac.Sum(nil)))
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	if w2.Code != http.StatusAccepted {
		t.Fatalf("expected 202 with valid signature, got %d (%s)", w2.Code, w2.Body.String())
	}
}

func TestUpdateAuthAPIRoundTrip(t *testing.T) {
	registerTestDispatchers()
	r, _, cleanup := setupRouter(t)
	defer cleanup()
	createSource(t, r, "API Signed", "api-signed")

	// Enable GitHub HMAC auth via the JSON API.
	body, _ := json.Marshal(map[string]any{
		"enabled": true, "preset": "github", "secret": "topsecret",
	})
	req := httptest.NewRequest(http.MethodPut, "/api/sources/api-signed/auth", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 saving auth, got %d (%s)", w.Code, w.Body.String())
	}
	var resp struct {
		AuthConfig struct {
			Enabled   bool   `json:"enabled"`
			Preset    string `json:"preset"`
			HasSecret bool   `json:"has_secret"`
		} `json:"auth_config"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if !resp.AuthConfig.Enabled || resp.AuthConfig.Preset != "github" || !resp.AuthConfig.HasSecret {
		t.Fatalf("unexpected sanitized auth in response: %+v", resp.AuthConfig)
	}
	if bytes.Contains(w.Body.Bytes(), []byte("topsecret")) {
		t.Fatalf("auth response leaked the secret: %s", w.Body.String())
	}

	// GET must also return the sanitized view, never the secret.
	getReq := httptest.NewRequest(http.MethodGet, "/api/sources/api-signed", nil)
	getW := httptest.NewRecorder()
	r.ServeHTTP(getW, getReq)
	if getW.Code != http.StatusOK {
		t.Fatalf("expected 200 fetching source, got %d", getW.Code)
	}
	if bytes.Contains(getW.Body.Bytes(), []byte("topsecret")) {
		t.Fatalf("source GET leaked the secret: %s", getW.Body.String())
	}

	// Ingest actually verifies with the stored secret.
	payload := []byte(`{"event":"push"}`)
	mac := hmac.New(sha256.New, []byte("topsecret"))
	mac.Write(payload)
	ingest := httptest.NewRequest(http.MethodPost, "/webhooks/api-signed", bytes.NewReader(payload))
	ingest.Header.Set("Content-Type", "application/json")
	ingest.Header.Set("X-Hub-Signature-256", "sha256="+hex.EncodeToString(mac.Sum(nil)))
	iw := httptest.NewRecorder()
	r.ServeHTTP(iw, ingest)
	if iw.Code != http.StatusAccepted {
		t.Fatalf("expected 202 with valid signature, got %d (%s)", iw.Code, iw.Body.String())
	}

	// Disabling clears the config; unsigned ingest is accepted again.
	offBody, _ := json.Marshal(map[string]any{"enabled": false})
	offReq := httptest.NewRequest(http.MethodPut, "/api/sources/api-signed/auth", bytes.NewReader(offBody))
	offReq.Header.Set("Content-Type", "application/json")
	offW := httptest.NewRecorder()
	r.ServeHTTP(offW, offReq)
	if offW.Code != http.StatusOK {
		t.Fatalf("expected 200 disabling auth, got %d (%s)", offW.Code, offW.Body.String())
	}
	plain := httptest.NewRequest(http.MethodPost, "/webhooks/api-signed", bytes.NewReader(payload))
	plain.Header.Set("Content-Type", "application/json")
	pw := httptest.NewRecorder()
	r.ServeHTTP(pw, plain)
	if pw.Code != http.StatusAccepted {
		t.Fatalf("expected 202 after disabling auth, got %d (%s)", pw.Code, pw.Body.String())
	}
}
