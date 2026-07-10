package inboundauth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"testing"
	"time"
)

func TestNoneSchemeAllows(t *testing.T) {
	cfg, err := ParseConfig(nil)
	if err != nil {
		t.Fatalf("parse nil: %v", err)
	}
	if cfg.Scheme != SchemeNone {
		t.Fatalf("expected none scheme, got %q", cfg.Scheme)
	}
	if err := Verify(cfg, http.Header{}, []byte("anything"), time.Unix(0, 0)); err != nil {
		t.Fatalf("none scheme should allow, got %v", err)
	}
}

func TestUnsupportedScheme(t *testing.T) {
	cfg := Config{Scheme: "banana"}
	if err := Verify(cfg, http.Header{}, nil, time.Unix(0, 0)); err != ErrUnsupportedScheme {
		t.Fatalf("expected ErrUnsupportedScheme, got %v", err)
	}
}

// GitHub's documented sample vector.
func TestHMACGitHubVector(t *testing.T) {
	cfg := Config{
		Scheme: SchemeHMAC, Algo: "sha256", Encoding: "hex",
		SigHeader: "X-Hub-Signature-256", SigParser: "plain",
		SigPrefix: "sha256=", Template: "raw_body",
		Secret: "It's a Secret to Everybody",
	}
	body := []byte("Hello, World!")
	h := http.Header{}
	h.Set("X-Hub-Signature-256", "sha256=757107ea0eb2509fc211221cce984b8a37570b6d7586c22c46f4379c8b043e17")
	if err := Verify(cfg, h, body, time.Unix(0, 0)); err != nil {
		t.Fatalf("expected valid GitHub signature, got %v", err)
	}
}

func TestHMACForgejoNoPrefix(t *testing.T) {
	secret, body := "forge-secret", []byte(`{"ref":"refs/heads/main"}`)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	sig := hex.EncodeToString(mac.Sum(nil))

	cfg := Config{
		Scheme: SchemeHMAC, Algo: "sha256", Encoding: "hex",
		SigHeader: "X-Forgejo-Signature", SigParser: "plain",
		Template: "raw_body", Secret: secret,
	}
	h := http.Header{}
	h.Set("X-Forgejo-Signature", sig)
	if err := Verify(cfg, h, body, time.Unix(0, 0)); err != nil {
		t.Fatalf("expected valid Forgejo signature, got %v", err)
	}

	// Tamper.
	if err := Verify(cfg, h, []byte("tampered"), time.Unix(0, 0)); err != ErrBadSignature {
		t.Fatalf("expected ErrBadSignature on tamper, got %v", err)
	}
	// Missing header.
	if err := Verify(cfg, http.Header{}, body, time.Unix(0, 0)); err != ErrMissingSignature {
		t.Fatalf("expected ErrMissingSignature, got %v", err)
	}
}
