package inboundauth

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestSanitizeStripsCredentials(t *testing.T) {
	raw := json.RawMessage(`{"scheme":"hmac","preset":"github","algo":"sha256","secret":"topsecret"}`)
	s := Sanitize(raw)
	if !s.Enabled || s.Scheme != SchemeHMAC || s.Preset != "github" {
		t.Fatalf("unexpected sanitized state: %+v", s)
	}
	if !s.HasSecret {
		t.Fatal("expected has_secret=true")
	}
	out, err := json.Marshal(s)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if strings.Contains(string(out), "topsecret") {
		t.Fatalf("sanitized output leaked the secret: %s", out)
	}
}

func TestSanitizeEmptyAndInvalid(t *testing.T) {
	if s := Sanitize(nil); s.Enabled || s.Scheme != SchemeNone || s.HasSecret {
		t.Fatalf("nil config should sanitize disabled, got %+v", s)
	}
	if s := Sanitize(json.RawMessage(`{not json`)); s.Enabled || s.Scheme != SchemeNone {
		t.Fatalf("invalid config should sanitize disabled, got %+v", s)
	}
}

func TestSanitizeTokenCountsAsSecret(t *testing.T) {
	raw := json.RawMessage(`{"scheme":"token","preset":"gitlab-token","token":"tok"}`)
	if s := Sanitize(raw); !s.HasSecret {
		t.Fatal("token scheme should report has_secret=true")
	}
}
