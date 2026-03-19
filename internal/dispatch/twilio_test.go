package dispatch

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/zachbroad/nitrohook/internal/model"
)

func TestTwilioDispatch(t *testing.T) {
	var capturedReq *http.Request
	var capturedBody []byte

	captureClient := &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			capturedReq = req.Clone(req.Context())
			capturedBody, _ = io.ReadAll(req.Body)
			return &http.Response{
				StatusCode: 201,
				Body:       io.NopCloser(strings.NewReader(`{"sid":"SM123"}`)),
				Header:     make(http.Header),
			}, nil
		}),
	}

	action := &model.Action{
		Type:   model.ActionTypeTwilio,
		Config: json.RawMessage(`{"account_sid":"AC123","auth_token":"tok456","from":"+15551234","to":"+15555678"}`),
	}

	d := &TwilioDispatcher{Client: captureClient}
	payload := json.RawMessage(`{"event":"test"}`)
	result := d.Dispatch(context.Background(), action, "del-1", payload, json.RawMessage(`{}`))

	if !result.Success {
		t.Fatalf("expected success, got error: %v", result.ErrorMessage)
	}

	// Verify Basic Auth
	user, pass, ok := capturedReq.BasicAuth()
	if !ok {
		t.Fatal("expected Basic Auth header")
	}
	if user != "AC123" || pass != "tok456" {
		t.Fatalf("expected auth AC123:tok456, got %s:%s", user, pass)
	}

	// Verify form body
	bodyStr := string(capturedBody)
	if !strings.Contains(bodyStr, "From=%2B15551234") {
		t.Fatalf("expected From in form body, got %q", bodyStr)
	}
	if !strings.Contains(bodyStr, "To=%2B15555678") {
		t.Fatalf("expected To in form body, got %q", bodyStr)
	}
	if !strings.Contains(bodyStr, "Body=") {
		t.Fatalf("expected Body in form body, got %q", bodyStr)
	}

	// Verify Content-Type
	if capturedReq.Header.Get("Content-Type") != "application/x-www-form-urlencoded" {
		t.Fatalf("expected form content type, got %q", capturedReq.Header.Get("Content-Type"))
	}

	// Verify URL contains account_sid
	if !strings.Contains(capturedReq.URL.String(), "AC123") {
		t.Fatalf("expected URL to contain account_sid, got %q", capturedReq.URL.String())
	}
}

func TestTwilioBodyTruncation(t *testing.T) {
	var capturedBody []byte

	captureClient := &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			capturedBody, _ = io.ReadAll(req.Body)
			return &http.Response{
				StatusCode: 201,
				Body:       io.NopCloser(strings.NewReader(`{"sid":"SM456"}`)),
				Header:     make(http.Header),
			}, nil
		}),
	}

	action := &model.Action{
		Type:   model.ActionTypeTwilio,
		Config: json.RawMessage(`{"account_sid":"AC123","auth_token":"tok","from":"+1111","to":"+2222"}`),
	}

	// Create a payload > 1600 chars
	longPayload := `{"data":"` + strings.Repeat("x", 2000) + `"}`

	d := &TwilioDispatcher{Client: captureClient}
	result := d.Dispatch(context.Background(), action, "del-2", json.RawMessage(longPayload), json.RawMessage(`{}`))

	if !result.Success {
		t.Fatalf("expected success, got error: %v", result.ErrorMessage)
	}

	// The Body form value should be truncated — URL-encoded "..." appears as "..."
	bodyStr := string(capturedBody)
	if !strings.Contains(bodyStr, "...") {
		t.Fatal("expected truncated body to end with ...")
	}
}

// roundTripFunc is an adapter to use a function as http.RoundTripper.
type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}
