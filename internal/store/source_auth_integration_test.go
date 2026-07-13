//go:build integration

package store_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/zachbroad/nitrohook/internal/testutil"
)

func TestSetAndGetAuthConfig(t *testing.T) {
	s, _ := testutil.SetupTestDB(t)
	ctx := context.Background()

	if _, err := s.Sources.Create(ctx, "Auth Src", "auth-src", "active", nil); err != nil {
		t.Fatalf("create source: %v", err)
	}

	// New sources start with no auth config.
	src, err := s.Sources.GetBySlug(ctx, "auth-src")
	if err != nil {
		t.Fatalf("get source: %v", err)
	}
	if src.AuthConfig != nil {
		t.Fatalf("expected nil auth_config, got %s", src.AuthConfig)
	}

	// Round-trip a config.
	cfg := json.RawMessage(`{"scheme":"hmac","secret":"s3cr3t"}`)
	if _, err := s.Sources.SetAuthConfig(ctx, "auth-src", cfg); err != nil {
		t.Fatalf("set auth config: %v", err)
	}
	got, err := s.Sources.GetBySlug(ctx, "auth-src")
	if err != nil {
		t.Fatalf("get after set: %v", err)
	}
	var m map[string]string
	if err := json.Unmarshal(got.AuthConfig, &m); err != nil {
		t.Fatalf("unmarshal stored config: %v", err)
	}
	if m["scheme"] != "hmac" || m["secret"] != "s3cr3t" {
		t.Fatalf("unexpected stored config: %v", m)
	}

	// Clearing sets it back to NULL.
	if _, err := s.Sources.SetAuthConfig(ctx, "auth-src", nil); err != nil {
		t.Fatalf("clear auth config: %v", err)
	}
	cleared, err := s.Sources.GetBySlug(ctx, "auth-src")
	if err != nil {
		t.Fatalf("get after clear: %v", err)
	}
	if cleared.AuthConfig != nil {
		t.Fatalf("expected cleared auth_config, got %s", cleared.AuthConfig)
	}
}
