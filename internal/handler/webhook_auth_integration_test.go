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
