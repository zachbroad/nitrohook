package dispatch

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/zachbroad/nitrohook/internal/model"
)

func TestSlackDispatch(t *testing.T) {
	var receivedBody []byte

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	}))
	defer server.Close()

	action := &model.Action{
		Type:   model.ActionTypeSlack,
		Config: json.RawMessage(`{"webhook_url":"` + server.URL + `"}`),
	}

	d := &SlackDispatcher{Client: server.Client()}
	payload := json.RawMessage(`{"event":"test"}`)
	result := d.Dispatch(context.Background(), action, "del-1", payload, json.RawMessage(`{}`))

	if !result.Success {
		t.Fatalf("expected success, got error: %v", result.ErrorMessage)
	}

	// Verify JSON body has "text" field
	var msg map[string]any
	if err := json.Unmarshal(receivedBody, &msg); err != nil {
		t.Fatalf("failed to parse slack message: %v", err)
	}
	text, ok := msg["text"].(string)
	if !ok || text == "" {
		t.Fatal("expected text field in slack message")
	}
	// The text should contain the payload
	if len(text) == 0 {
		t.Fatal("expected non-empty text")
	}
}

func TestSlackDispatchWithChannel(t *testing.T) {
	var receivedBody []byte

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	action := &model.Action{
		Type:   model.ActionTypeSlack,
		Config: json.RawMessage(`{"webhook_url":"` + server.URL + `","channel":"#alerts","username":"webhook-bot"}`),
	}

	d := &SlackDispatcher{Client: server.Client()}
	result := d.Dispatch(context.Background(), action, "del-2", json.RawMessage(`{}`), json.RawMessage(`{}`))

	if !result.Success {
		t.Fatalf("expected success, got error: %v", result.ErrorMessage)
	}

	var msg map[string]any
	if err := json.Unmarshal(receivedBody, &msg); err != nil {
		t.Fatalf("failed to parse slack message: %v", err)
	}
	if msg["channel"] != "#alerts" {
		t.Fatalf("expected channel #alerts, got %v", msg["channel"])
	}
	if msg["username"] != "webhook-bot" {
		t.Fatalf("expected username webhook-bot, got %v", msg["username"])
	}
}

func TestSlackDispatchFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte("invalid_token"))
	}))
	defer server.Close()

	action := &model.Action{
		Type:   model.ActionTypeSlack,
		Config: json.RawMessage(`{"webhook_url":"` + server.URL + `"}`),
	}

	d := &SlackDispatcher{Client: server.Client()}
	result := d.Dispatch(context.Background(), action, "del-3", json.RawMessage(`{}`), json.RawMessage(`{}`))

	if result.Success {
		t.Fatal("expected failure for 403 response")
	}
	if result.ErrorMessage == nil {
		t.Fatal("expected error message")
	}
}
