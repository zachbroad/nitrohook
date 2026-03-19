package dispatch

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/zachbroad/nitrohook/internal/model"
	"github.com/zachbroad/nitrohook/internal/signing"
)

func TestWebhookDispatch(t *testing.T) {
	var receivedBody []byte
	var receivedHeaders http.Header

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedHeaders = r.Header
		receivedBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	}))
	defer server.Close()

	targetURL := server.URL
	secret := "test-secret"
	action := &model.Action{
		Type:          model.ActionTypeWebhook,
		TargetURL:     &targetURL,
		SigningSecret: &secret,
	}

	d := &WebhookDispatcher{Client: server.Client()}
	payload := json.RawMessage(`{"event":"push"}`)
	headers := json.RawMessage(`{"X-Request-ID":"req-123"}`)

	result := d.Dispatch(context.Background(), action, "delivery-1", payload, headers)

	if !result.Success {
		t.Fatalf("expected success, got error: %v", result.ErrorMessage)
	}
	if result.ResponseStatus == nil || *result.ResponseStatus != 200 {
		t.Fatalf("expected status 200, got %v", result.ResponseStatus)
	}

	// Verify payload forwarded
	if string(receivedBody) != `{"event":"push"}` {
		t.Fatalf("expected payload forwarded, got %q", string(receivedBody))
	}

	// Verify headers forwarded
	if receivedHeaders.Get("X-Request-ID") != "req-123" {
		t.Fatalf("expected X-Request-ID header, got %q", receivedHeaders.Get("X-Request-ID"))
	}
	if receivedHeaders.Get("X-Delivery-ID") != "delivery-1" {
		t.Fatalf("expected X-Delivery-ID header, got %q", receivedHeaders.Get("X-Delivery-ID"))
	}
	if receivedHeaders.Get("Content-Type") != "application/json" {
		t.Fatalf("expected Content-Type application/json, got %q", receivedHeaders.Get("Content-Type"))
	}

	// Verify HMAC signature
	sig := receivedHeaders.Get("X-Webhook-Signature-256")
	if sig == "" {
		t.Fatal("expected HMAC signature header")
	}
	if !signing.Verify(payload, secret, sig) {
		t.Fatal("HMAC signature verification failed")
	}
}

func TestWebhookDispatchNoSigning(t *testing.T) {
	var receivedHeaders http.Header

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedHeaders = r.Header
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	targetURL := server.URL
	action := &model.Action{
		Type:      model.ActionTypeWebhook,
		TargetURL: &targetURL,
	}

	d := &WebhookDispatcher{Client: server.Client()}
	result := d.Dispatch(context.Background(), action, "delivery-2", json.RawMessage(`{}`), json.RawMessage(`{}`))

	if !result.Success {
		t.Fatalf("expected success, got error: %v", result.ErrorMessage)
	}
	if receivedHeaders.Get("X-Webhook-Signature-256") != "" {
		t.Fatal("expected no signature header when signing_secret is nil")
	}
}

func TestWebhookDispatchFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("server error"))
	}))
	defer server.Close()

	targetURL := server.URL
	action := &model.Action{
		Type:      model.ActionTypeWebhook,
		TargetURL: &targetURL,
	}

	d := &WebhookDispatcher{Client: server.Client()}
	result := d.Dispatch(context.Background(), action, "delivery-3", json.RawMessage(`{}`), json.RawMessage(`{}`))

	if result.Success {
		t.Fatal("expected failure for 500 response")
	}
	if result.ResponseStatus == nil || *result.ResponseStatus != 500 {
		t.Fatalf("expected status 500, got %v", result.ResponseStatus)
	}
	if result.ErrorMessage == nil {
		t.Fatal("expected error message")
	}
}
