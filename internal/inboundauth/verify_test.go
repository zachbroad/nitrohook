package inboundauth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"net/http"
	"strconv"
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

func hmacHex(secret, msg string) string {
	m := hmac.New(sha256.New, []byte(secret))
	m.Write([]byte(msg))
	return hex.EncodeToString(m.Sum(nil))
}

func TestHMACStripeTimestamped(t *testing.T) {
	secret := "whsec_test"
	body := []byte(`{"id":"evt_1"}`)
	ts := int64(1_700_000_000)
	sig := hmacHex(secret, strconv.FormatInt(ts, 10)+"."+string(body))

	cfg := Config{
		Scheme: SchemeHMAC, Algo: "sha256", Encoding: "hex",
		SigHeader: "Stripe-Signature", SigParser: "kv-comma",
		Template: "ts.body", TSToleranceS: 300, Secret: secret,
	}
	h := http.Header{}
	h.Set("Stripe-Signature", "t="+strconv.FormatInt(ts, 10)+",v1="+sig)

	now := time.Unix(ts+10, 0)
	if err := Verify(cfg, h, body, now); err != nil {
		t.Fatalf("valid stripe sig rejected: %v", err)
	}
	// Outside tolerance.
	if err := Verify(cfg, h, body, time.Unix(ts+1000, 0)); err != ErrTimestampOutOfTolerance {
		t.Fatalf("expected tolerance error, got %v", err)
	}
}

func TestHMACSlack(t *testing.T) {
	secret := "slack_secret"
	body := []byte("token=xyz&team_id=T1")
	ts := int64(1_700_000_100)
	sig := "v0=" + hmacHex(secret, "v0:"+strconv.FormatInt(ts, 10)+":"+string(body))

	cfg := Config{
		Scheme: SchemeHMAC, Algo: "sha256", Encoding: "hex",
		SigHeader: "X-Slack-Signature", SigParser: "slack",
		Template: "slack_v0", TSHeader: "X-Slack-Request-Timestamp",
		TSToleranceS: 300, Secret: secret,
	}
	h := http.Header{}
	h.Set("X-Slack-Signature", sig)
	h.Set("X-Slack-Request-Timestamp", strconv.FormatInt(ts, 10))
	if err := Verify(cfg, h, body, time.Unix(ts+5, 0)); err != nil {
		t.Fatalf("valid slack sig rejected: %v", err)
	}
}

func TestHMACSvixSpaceList(t *testing.T) {
	secret := "svix_secret"
	body := []byte(`{"a":1}`)
	id, ts := "msg_1", int64(1_700_000_200)
	msg := id + "." + strconv.FormatInt(ts, 10) + "." + string(body)
	m := hmac.New(sha256.New, []byte(secret))
	m.Write([]byte(msg))
	b64 := base64.StdEncoding.EncodeToString(m.Sum(nil))

	cfg := Config{
		Scheme: SchemeHMAC, Algo: "sha256", Encoding: "base64",
		SigHeader: "webhook-signature", SigParser: "space-list",
		Template: "id.ts.body", TSHeader: "webhook-timestamp",
		IDHeader: "webhook-id", TSToleranceS: 300, Secret: secret,
	}
	h := http.Header{}
	h.Set("webhook-id", id)
	h.Set("webhook-timestamp", strconv.FormatInt(ts, 10))
	h.Set("webhook-signature", "v1,"+b64+" v1,AAAAAAAA") // multiple space-separated tokens
	if err := Verify(cfg, h, body, time.Unix(ts+5, 0)); err != nil {
		t.Fatalf("valid svix sig rejected: %v", err)
	}
}

func TestTokenScheme(t *testing.T) {
	cfg := Config{Scheme: SchemeToken, TokenHdr: "X-Gitlab-Token", Token: "sekret"}
	h := http.Header{}
	h.Set("X-Gitlab-Token", "sekret")
	if err := Verify(cfg, h, nil, time.Unix(0, 0)); err != nil {
		t.Fatalf("valid token rejected: %v", err)
	}
	h.Set("X-Gitlab-Token", "wrong")
	if err := Verify(cfg, h, nil, time.Unix(0, 0)); err != ErrBadSignature {
		t.Fatalf("expected ErrBadSignature, got %v", err)
	}
	if err := Verify(cfg, http.Header{}, nil, time.Unix(0, 0)); err != ErrMissingSignature {
		t.Fatalf("expected ErrMissingSignature, got %v", err)
	}
}

func TestBearerScheme(t *testing.T) {
	cfg := Config{Scheme: SchemeBearer, Token: "abc123"}
	h := http.Header{}
	h.Set("Authorization", "Bearer abc123")
	if err := Verify(cfg, h, nil, time.Unix(0, 0)); err != nil {
		t.Fatalf("valid bearer rejected: %v", err)
	}
	h.Set("Authorization", "Bearer nope")
	if err := Verify(cfg, h, nil, time.Unix(0, 0)); err != ErrBadSignature {
		t.Fatalf("expected ErrBadSignature, got %v", err)
	}
}
