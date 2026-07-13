package inboundauth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"testing"
	"time"
)

func TestPresetGitHubResolvesAndVerifies(t *testing.T) {
	cfg, ok := Preset("github")
	if !ok {
		t.Fatal("github preset missing")
	}
	cfg.Secret = "It's a Secret to Everybody"
	h := http.Header{}
	h.Set("X-Hub-Signature-256", "sha256=757107ea0eb2509fc211221cce984b8a37570b6d7586c22c46f4379c8b043e17")
	if err := Verify(cfg, h, []byte("Hello, World!"), time.Unix(0, 0)); err != nil {
		t.Fatalf("github preset failed to verify documented vector: %v", err)
	}
}

func TestPresetForgejoRoundTrip(t *testing.T) {
	cfg, ok := Preset("forgejo")
	if !ok {
		t.Fatal("forgejo preset missing")
	}
	cfg.Secret = "s"
	body := []byte(`{"x":1}`)
	m := hmac.New(sha256.New, []byte("s"))
	m.Write(body)
	h := http.Header{}
	h.Set("X-Forgejo-Signature", hex.EncodeToString(m.Sum(nil)))
	if err := Verify(cfg, h, body, time.Unix(0, 0)); err != nil {
		t.Fatalf("forgejo preset failed: %v", err)
	}
}

func TestPresetNamesCoverage(t *testing.T) {
	want := []string{"github", "forgejo", "stripe", "slack", "shopify", "standard-webhooks", "gitlab-token", "discord-ed25519"}
	for _, name := range want {
		if _, ok := Preset(name); !ok {
			t.Errorf("missing preset %q", name)
		}
	}
	if len(PresetNames()) != len(want) {
		t.Errorf("PresetNames count = %d, want %d", len(PresetNames()), len(want))
	}
}
