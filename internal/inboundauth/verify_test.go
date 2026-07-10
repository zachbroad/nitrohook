package inboundauth

import (
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
