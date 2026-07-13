# Incoming Webhook Authentication Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Verify incoming webhooks against per-source auth config before anything is persisted or published, rejecting unauthenticated requests with `401`, supporting the signature schemes used by the major webhook platforms.

**Architecture:** A new pure package `internal/inboundauth` provides a preset-driven, parameterized verifier (`Verify(cfg, header, rawBody, now) error`). Sources carry an `auth_config` JSONB column decoded into that config. The ingest handler runs verification between body-read and delivery-persist. A dedicated `SetAuthConfig` store method + `UpdateSourceAuth` web handler manage config through an htmx "auth-card" fragment, mirroring the existing "mode-card" pattern.

**Tech Stack:** Go, Gin, pgx/PostgreSQL, golang-migrate, Prometheus (promauto), htmx templates, Go stdlib crypto (`crypto/hmac`, `crypto/sha1`, `crypto/sha256`, `crypto/ed25519`, `crypto/subtle`).

## Global Constraints

- Backward compatibility: a source with `NULL`/absent `auth_config` MUST behave exactly as today (scheme `none`, no verification). Verified by an explicit test.
- All secret/signature comparisons MUST be constant-time (`hmac.Equal` or `crypto/subtle.ConstantTimeCompare`).
- Verification runs on the **raw request bytes** (`io.ReadAll` output), before `Deliveries.Create`. Nothing is persisted or published on failure.
- Every rejection increments `metrics.WebhookAuthFailures` with labels `{source, reason}` and returns HTTP `401`.
- Secrets are stored **plaintext** inside the JSONB (matches `actions.signing_secret`). No encryption in this PR.
- Unit tests: plain package, no build tag (like `internal/signing/hmac_test.go`). DB/Redis tests: `//go:build integration`, use `internal/testutil`.
- Next migration number is **000009** (last is `000008_add_action_transform_script`).
- Presets in scope: `github`, `forgejo`, `stripe`, `slack`, `shopify`, `standard-webhooks`, `gitlab-token`, `discord-ed25519`, plus `custom`. Twilio and Basic auth are explicitly out of scope.

---

## File Structure

**Create:**
- `migrations/000009_add_source_auth.up.sql` / `.down.sql` — `auth_config JSONB` on `sources`.
- `internal/inboundauth/config.go` — `Scheme`, `Config`, `ParseConfig`, typed errors, `FailureReason`.
- `internal/inboundauth/verify.go` — `Verify` dispatch + HMAC/token/bearer/ed25519 engines + header parsers.
- `internal/inboundauth/presets.go` — `Preset(name)`, `PresetNames`.
- `internal/inboundauth/verify_test.go`, `internal/inboundauth/presets_test.go` — unit tests.
- `web/templates/` auth-card fragment (added inside `source-overview.html`).
- `docs/src/content/docs/guides/authentication.md` — user guide.

**Modify:**
- `internal/model/model.go` — `Source.AuthConfig json.RawMessage`.
- `internal/store/source.go` — add `auth_config` to all SELECT column lists + `Scan`; add `SetAuthConfig`.
- `internal/metrics/metrics.go` — `WebhookAuthFailures` counter.
- `internal/handler/webhook.go` — verification gate.
- `web/source.go` — `UpdateSourceAuth` handler + `sourceData` auth fields.
- `web/templates/source-overview.html` — render + define `auth-card`.
- `cmd/api/main.go` — `POST /sources/:slug/auth` route.

---

## Task 1: Persist `auth_config` on sources

**Files:**
- Create: `migrations/000009_add_source_auth.up.sql`, `migrations/000009_add_source_auth.down.sql`
- Modify: `internal/model/model.go:10-18`
- Modify: `internal/store/source.go` (all SELECT lists + `Scan` calls; add `SetAuthConfig`)
- Test: `internal/store/source_auth_integration_test.go`

**Interfaces:**
- Produces: `model.Source.AuthConfig json.RawMessage`; `func (s *SourceStore) SetAuthConfig(ctx context.Context, slug string, cfg json.RawMessage) (*model.Source, error)`.

- [ ] **Step 1: Write the migration files**

`migrations/000009_add_source_auth.up.sql`:
```sql
ALTER TABLE sources ADD COLUMN auth_config JSONB;
```

`migrations/000009_add_source_auth.down.sql`:
```sql
ALTER TABLE sources DROP COLUMN auth_config;
```

- [ ] **Step 2: Add the model field**

In `internal/model/model.go`, change the `Source` struct to:
```go
type Source struct {
	ID         uuid.UUID       `json:"id"`
	Name       string          `json:"name"`
	Slug       string          `json:"slug"`
	Mode       string          `json:"mode"`
	ScriptBody *string         `json:"script_body,omitempty"`
	AuthConfig json.RawMessage `json:"auth_config,omitempty"`
	CreatedAt  time.Time       `json:"created_at"`
	UpdatedAt  time.Time       `json:"updated_at"`
}
```
(`encoding/json` is already imported in this file.)

- [ ] **Step 3: Thread `auth_config` through the store reads**

In `internal/store/source.go`, update **every** SELECT column list from
`id, name, slug, mode, script_body, created_at, updated_at` to
`id, name, slug, mode, script_body, auth_config, created_at, updated_at`,
and add `&src.AuthConfig` to the matching position in **every** `.Scan(...)` call
(after `&src.ScriptBody`, before `&src.CreatedAt`). This applies to `GetBySlug`,
`GetByID`, `List`, `Create` (its RETURNING clause), and both branches of `Update`.

Example for `GetBySlug` (lines 18-28):
```go
func (s *SourceStore) GetBySlug(ctx context.Context, slug string) (*model.Source, error) {
	var src model.Source
	err := s.pool.QueryRow(ctx,
		`SELECT id, name, slug, mode, script_body, auth_config, created_at, updated_at FROM sources WHERE slug = $1`,
		slug,
	).Scan(&src.ID, &src.Name, &src.Slug, &src.Mode, &src.ScriptBody, &src.AuthConfig, &src.CreatedAt, &src.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("get source by slug: %w", err)
	}
	return &src, nil
}
```
Apply the identical column-list + `&src.AuthConfig` change to `GetByID`, `List`, `Create`, and both `Update` branches.

- [ ] **Step 4: Add the `SetAuthConfig` write method**

Append to `internal/store/source.go`:
```go
// SetAuthConfig replaces a source's auth_config. Passing nil clears it (scheme "none").
func (s *SourceStore) SetAuthConfig(ctx context.Context, slug string, cfg json.RawMessage) (*model.Source, error) {
	var src model.Source
	err := s.pool.QueryRow(ctx,
		`UPDATE sources SET auth_config = $2, updated_at = $3
		 WHERE slug = $1
		 RETURNING id, name, slug, mode, script_body, auth_config, created_at, updated_at`,
		slug, cfg, time.Now(),
	).Scan(&src.ID, &src.Name, &src.Slug, &src.Mode, &src.ScriptBody, &src.AuthConfig, &src.CreatedAt, &src.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("source not found")
		}
		return nil, fmt.Errorf("set auth config: %w", err)
	}
	return &src, nil
}
```
Add `"encoding/json"` to the import block of `internal/store/source.go`.

- [ ] **Step 5: Write the failing integration test**

Create `internal/store/source_auth_integration_test.go`:
```go
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
	cleared, _ := s.Sources.GetBySlug(ctx, "auth-src")
	if cleared.AuthConfig != nil {
		t.Fatalf("expected cleared auth_config, got %s", cleared.AuthConfig)
	}
}
```

- [ ] **Step 6: Run migrations and the test to verify it passes**

Run:
```bash
make docker-up-supporting-svc
make migrate-up
go test -tags=integration -run TestSetAndGetAuthConfig ./internal/store/...
```
Expected: PASS. Also run `go build ./...` to confirm all `Scan` call sites still compile.

- [ ] **Step 7: Commit**

```bash
git add migrations/000009_add_source_auth.up.sql migrations/000009_add_source_auth.down.sql \
        internal/model/model.go internal/store/source.go internal/store/source_auth_integration_test.go
git commit -m "feat(auth): persist auth_config on sources"
```

---

## Task 2: `inboundauth` config, dispatch skeleton, and `none` scheme

**Files:**
- Create: `internal/inboundauth/config.go`
- Create: `internal/inboundauth/verify.go`
- Test: `internal/inboundauth/verify_test.go`

**Interfaces:**
- Produces:
  - `type Scheme string` with `SchemeNone/SchemeHMAC/SchemeBearer/SchemeToken/SchemeEd25519`.
  - `type Config struct { ... }` (fields below).
  - `func ParseConfig(raw json.RawMessage) (Config, error)` — nil/empty ⇒ `Config{Scheme: SchemeNone}`.
  - `func Verify(cfg Config, header http.Header, rawBody []byte, now time.Time) error`.
  - Sentinel errors `ErrMissingSignature, ErrBadSignature, ErrMissingTimestamp, ErrTimestampOutOfTolerance, ErrUnsupportedScheme`.
  - `func FailureReason(err error) string` — maps an error to a short metric label.

- [ ] **Step 1: Write the failing test**

Create `internal/inboundauth/verify_test.go`:
```go
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
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/inboundauth/...`
Expected: FAIL (package/identifiers undefined).

- [ ] **Step 3: Write `config.go`**

Create `internal/inboundauth/config.go`:
```go
// Package inboundauth verifies the authenticity of incoming webhook requests
// against per-source configuration covering the major provider signing schemes.
package inboundauth

import (
	"encoding/json"
	"errors"
)

type Scheme string

const (
	SchemeNone    Scheme = "none"
	SchemeHMAC    Scheme = "hmac"
	SchemeBearer  Scheme = "bearer"
	SchemeToken   Scheme = "token"
	SchemeEd25519 Scheme = "ed25519"
)

// Config is the decoded per-source auth configuration (from sources.auth_config).
type Config struct {
	Scheme Scheme `json:"scheme"`
	Preset string `json:"preset,omitempty"`

	// HMAC axes.
	Algo      string `json:"algo,omitempty"`       // "sha256" | "sha1"
	Encoding  string `json:"encoding,omitempty"`   // "hex" | "base64"
	SigHeader string `json:"sig_header,omitempty"` // header carrying the signature
	SigParser string `json:"sig_parser,omitempty"` // "plain" | "kv-comma" | "space-list" | "slack"
	SigPrefix string `json:"sig_prefix,omitempty"` // stripped by the "plain" parser (e.g. "sha256=")
	Template  string `json:"template,omitempty"`   // "raw_body" | "ts.body" | "id.ts.body" | "slack_v0"

	// Timestamp / replay (used by ts-bound templates).
	TSHeader     string `json:"ts_header,omitempty"`
	IDHeader     string `json:"id_header,omitempty"`
	TSToleranceS int    `json:"ts_tolerance_secs,omitempty"`

	// Secrets (plaintext at rest).
	Secret    string `json:"secret,omitempty"`     // HMAC shared secret
	Token     string `json:"token,omitempty"`      // bearer / plain-token expected value
	TokenHdr  string `json:"token_header,omitempty"` // header for "token" scheme (e.g. X-Gitlab-Token)
	PublicKey string `json:"public_key,omitempty"` // ed25519 public key, hex-encoded
}

var (
	ErrMissingSignature        = errors.New("missing signature header")
	ErrBadSignature            = errors.New("signature mismatch")
	ErrMissingTimestamp        = errors.New("missing timestamp header")
	ErrTimestampOutOfTolerance = errors.New("timestamp outside tolerance window")
	ErrUnsupportedScheme       = errors.New("unsupported auth scheme")
)

// ParseConfig decodes sources.auth_config. Nil/empty means no authentication.
func ParseConfig(raw json.RawMessage) (Config, error) {
	if len(raw) == 0 {
		return Config{Scheme: SchemeNone}, nil
	}
	var cfg Config
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return Config{}, err
	}
	if cfg.Scheme == "" {
		cfg.Scheme = SchemeNone
	}
	return cfg, nil
}

// FailureReason maps a verification error to a short, low-cardinality metric label.
func FailureReason(err error) string {
	switch {
	case errors.Is(err, ErrMissingSignature):
		return "missing_signature"
	case errors.Is(err, ErrBadSignature):
		return "bad_signature"
	case errors.Is(err, ErrMissingTimestamp):
		return "missing_timestamp"
	case errors.Is(err, ErrTimestampOutOfTolerance):
		return "timestamp"
	case errors.Is(err, ErrUnsupportedScheme):
		return "unsupported"
	default:
		return "error"
	}
}
```

- [ ] **Step 4: Write `verify.go` dispatch skeleton**

Create `internal/inboundauth/verify.go`:
```go
package inboundauth

import (
	"net/http"
	"time"
)

// Verify returns nil if the request is authentic under cfg, or a sentinel error.
func Verify(cfg Config, header http.Header, rawBody []byte, now time.Time) error {
	switch cfg.Scheme {
	case SchemeNone:
		return nil
	case SchemeHMAC:
		return verifyHMAC(cfg, header, rawBody, now)
	case SchemeToken:
		return verifyToken(cfg, header)
	case SchemeBearer:
		return verifyBearer(cfg, header)
	case SchemeEd25519:
		return verifyEd25519(cfg, header, rawBody, now)
	default:
		return ErrUnsupportedScheme
	}
}
```
(The `verifyHMAC/verifyToken/verifyBearer/verifyEd25519` functions are added in Tasks 3-6. To compile now, add temporary stubs at the bottom of `verify.go` that each `return ErrUnsupportedScheme`; each subsequent task replaces its stub.)

Temporary stubs to append:
```go
func verifyHMAC(Config, http.Header, []byte, time.Time) error { return ErrUnsupportedScheme }
func verifyToken(Config, http.Header) error                   { return ErrUnsupportedScheme }
func verifyBearer(Config, http.Header) error                  { return ErrUnsupportedScheme }
func verifyEd25519(Config, http.Header, []byte, time.Time) error { return ErrUnsupportedScheme }
```

- [ ] **Step 5: Run test to verify it passes**

Run: `go test ./internal/inboundauth/...`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/inboundauth/config.go internal/inboundauth/verify.go internal/inboundauth/verify_test.go
git commit -m "feat(auth): inboundauth config, dispatch, and none scheme"
```

---

## Task 3: HMAC engine — raw-body plain parser (GitHub, Forgejo, Shopify)

**Files:**
- Modify: `internal/inboundauth/verify.go` (replace `verifyHMAC` stub + helpers)
- Test: `internal/inboundauth/verify_test.go`

**Interfaces:**
- Consumes: `Config`, sentinel errors from Task 2.
- Produces: working `verifyHMAC` for `Template == "raw_body"` with `SigParser == "plain"`, `Algo` ∈ {sha256, sha1}, `Encoding` ∈ {hex, base64}, optional `SigPrefix` strip.

- [ ] **Step 1: Write the failing tests**

Append to `internal/inboundauth/verify_test.go`:
```go
import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
)

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
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test -run TestHMAC ./internal/inboundauth/...`
Expected: FAIL (stub returns `ErrUnsupportedScheme`).

- [ ] **Step 3: Implement the HMAC engine**

In `internal/inboundauth/verify.go`, replace the `verifyHMAC` stub and add helpers. Update imports:
```go
import (
	"crypto/hmac"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"hash"
	"net/http"
	"strings"
	"time"
)
```
Compare by decoding the received signature to raw bytes per the configured
encoding, then `hmac.Equal` on the bytes. **Do not lowercase and string-compare:**
base64 is case-sensitive, so lowercasing both sides can make two different
signatures collide (a false accept). Hex decoding is already case-insensitive.
```go
func verifyHMAC(cfg Config, header http.Header, rawBody []byte, now time.Time) error {
	sig, err := extractSignature(cfg, header)
	if err != nil {
		return err
	}
	signed, err := buildSignedPayload(cfg, header, rawBody, now)
	if err != nil {
		return err
	}
	expected := computeHMAC(cfg, signed)
	got, err := decodeSig(cfg, sig)
	if err != nil {
		return ErrBadSignature
	}
	if !hmac.Equal(expected, got) {
		return ErrBadSignature
	}
	return nil
}

func hasher(algo string) func() hash.Hash {
	if algo == "sha1" {
		return sha1.New
	}
	return sha256.New
}

// computeHMAC returns the raw (un-encoded) HMAC of signed.
func computeHMAC(cfg Config, signed []byte) []byte {
	mac := hmac.New(hasher(cfg.Algo), []byte(cfg.Secret))
	mac.Write(signed)
	return mac.Sum(nil)
}

// decodeSig decodes a received signature string into raw bytes per cfg.Encoding.
func decodeSig(cfg Config, sig string) ([]byte, error) {
	if cfg.Encoding == "base64" {
		return base64.StdEncoding.DecodeString(sig)
	}
	return hex.DecodeString(sig)
}

// extractSignature returns the raw signature value using the configured parser.
// The "plain" parser handles the raw-body family (GitHub, Forgejo, Shopify).
// Other parsers are added in Task 4.
func extractSignature(cfg Config, header http.Header) (string, error) {
	switch cfg.SigParser {
	case "plain", "":
		raw := header.Get(cfg.SigHeader)
		if raw == "" {
			return "", ErrMissingSignature
		}
		return strings.TrimPrefix(raw, cfg.SigPrefix), nil
	default:
		return extractSignatureAdvanced(cfg, header)
	}
}

// buildSignedPayload constructs the byte string that gets HMAC'd.
// "raw_body" is the whole body. Other templates are added in Task 4.
func buildSignedPayload(cfg Config, header http.Header, rawBody []byte, now time.Time) ([]byte, error) {
	switch cfg.Template {
	case "raw_body", "":
		return rawBody, nil
	default:
		return buildSignedPayloadAdvanced(cfg, header, rawBody, now)
	}
}
```
Add temporary stubs (replaced in Task 4) so it compiles:
```go
func extractSignatureAdvanced(Config, http.Header) (string, error) { return "", ErrUnsupportedScheme }
func buildSignedPayloadAdvanced(Config, http.Header, []byte, time.Time) ([]byte, error) {
	return nil, ErrUnsupportedScheme
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/inboundauth/...`
Expected: PASS (GitHub vector + Forgejo + tamper + missing).

- [ ] **Step 5: Commit**

```bash
git add internal/inboundauth/verify.go internal/inboundauth/verify_test.go
git commit -m "feat(auth): HMAC verification for raw-body schemes"
```

---

## Task 4: HMAC engine — timestamped templates (Stripe, Slack, Svix)

**Files:**
- Modify: `internal/inboundauth/verify.go` (replace the two `*Advanced` stubs)
- Test: `internal/inboundauth/verify_test.go`

**Interfaces:**
- Consumes: HMAC engine from Task 3.
- Produces: `extractSignatureAdvanced` (parsers `kv-comma`, `space-list`, `slack`) and `buildSignedPayloadAdvanced` (templates `ts.body`, `id.ts.body`, `slack_v0`) with tolerance enforcement.

- [ ] **Step 1: Write the failing tests**

Append to `internal/inboundauth/verify_test.go`:
```go
import "strconv"

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
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test -run TestHMAC ./internal/inboundauth/...`
Expected: FAIL (advanced stubs return `ErrUnsupportedScheme`).

- [ ] **Step 3: Implement advanced parsers and templates**

In `internal/inboundauth/verify.go`, replace the two `*Advanced` stubs. Add `"strconv"` to imports. For `space-list` and `kv-comma`, multiple candidate signatures may appear; compare each in constant time and accept any match. Rework `verifyHMAC`'s compare so multi-candidate parsers are handled: change `extractSignature` to return `[]string`.

Replace the `verifyHMAC` compare block and `extractSignature` signature. Keep the
decode-then-`hmac.Equal` comparison from Task 3, applied to each candidate:
```go
func verifyHMAC(cfg Config, header http.Header, rawBody []byte, now time.Time) error {
	sigs, err := extractSignatures(cfg, header)
	if err != nil {
		return err
	}
	signed, err := buildSignedPayload(cfg, header, rawBody, now)
	if err != nil {
		return err
	}
	expected := computeHMAC(cfg, signed)
	for _, sig := range sigs {
		got, err := decodeSig(cfg, sig)
		if err != nil {
			continue
		}
		if hmac.Equal(expected, got) {
			return nil
		}
	}
	return ErrBadSignature
}

func extractSignatures(cfg Config, header http.Header) ([]string, error) {
	switch cfg.SigParser {
	case "plain", "":
		raw := header.Get(cfg.SigHeader)
		if raw == "" {
			return nil, ErrMissingSignature
		}
		return []string{strings.TrimPrefix(raw, cfg.SigPrefix)}, nil
	case "slack":
		raw := header.Get(cfg.SigHeader)
		if raw == "" {
			return nil, ErrMissingSignature
		}
		return []string{strings.TrimPrefix(raw, "v0=")}, nil
	case "kv-comma": // Stripe: "t=...,v1=...,v0=..."
		raw := header.Get(cfg.SigHeader)
		if raw == "" {
			return nil, ErrMissingSignature
		}
		var out []string
		for _, part := range strings.Split(raw, ",") {
			kv := strings.SplitN(strings.TrimSpace(part), "=", 2)
			if len(kv) == 2 && kv[0] == "v1" {
				out = append(out, kv[1])
			}
		}
		if len(out) == 0 {
			return nil, ErrMissingSignature
		}
		return out, nil
	case "space-list": // Svix: "v1,<b64> v1,<b64>"
		raw := header.Get(cfg.SigHeader)
		if raw == "" {
			return nil, ErrMissingSignature
		}
		var out []string
		for _, tok := range strings.Fields(raw) {
			if i := strings.IndexByte(tok, ','); i >= 0 {
				out = append(out, tok[i+1:])
			}
		}
		if len(out) == 0 {
			return nil, ErrMissingSignature
		}
		return out, nil
	default:
		return nil, ErrUnsupportedScheme
	}
}
```
Delete the now-unused `extractSignature`/`extractSignatureAdvanced` functions. Then replace `buildSignedPayload`/`buildSignedPayloadAdvanced` with:
```go
func buildSignedPayload(cfg Config, header http.Header, rawBody []byte, now time.Time) ([]byte, error) {
	switch cfg.Template {
	case "raw_body", "":
		return rawBody, nil
	case "ts.body", "id.ts.body", "slack_v0":
		ts := header.Get(cfg.TSHeader)
		if ts == "" {
			return nil, ErrMissingTimestamp
		}
		if err := checkTolerance(ts, cfg.TSToleranceS, now); err != nil {
			return nil, err
		}
		switch cfg.Template {
		case "ts.body":
			return []byte(ts + "." + string(rawBody)), nil
		case "slack_v0":
			return []byte("v0:" + ts + ":" + string(rawBody)), nil
		case "id.ts.body":
			id := header.Get(cfg.IDHeader)
			return []byte(id + "." + ts + "." + string(rawBody)), nil
		}
	}
	return nil, ErrUnsupportedScheme
}

func checkTolerance(tsStr string, toleranceS int, now time.Time) error {
	if toleranceS <= 0 {
		return nil
	}
	ts, err := strconv.ParseInt(tsStr, 10, 64)
	if err != nil {
		return ErrMissingTimestamp
	}
	diff := now.Unix() - ts
	if diff < 0 {
		diff = -diff
	}
	if diff > int64(toleranceS) {
		return ErrTimestampOutOfTolerance
	}
	return nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/inboundauth/...`
Expected: PASS (Stripe, Slack, Svix, plus Task 3 tests still green).

- [ ] **Step 5: Commit**

```bash
git add internal/inboundauth/verify.go internal/inboundauth/verify_test.go
git commit -m "feat(auth): HMAC verification for timestamped schemes"
```

---

## Task 5: Equality schemes — token (GitLab) and bearer

**Files:**
- Modify: `internal/inboundauth/verify.go` (replace `verifyToken`, `verifyBearer` stubs)
- Test: `internal/inboundauth/verify_test.go`

**Interfaces:**
- Produces: `verifyToken` (compares `cfg.TokenHdr` header to `cfg.Token`), `verifyBearer` (compares `Authorization: Bearer <cfg.Token>`), both constant-time.

- [ ] **Step 1: Write the failing tests**

Append to `internal/inboundauth/verify_test.go`:
```go
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
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test -run 'TestToken|TestBearer' ./internal/inboundauth/...`
Expected: FAIL.

- [ ] **Step 3: Implement the equality schemes**

Add `"crypto/subtle"` to the imports of `internal/inboundauth/verify.go` (first use), then replace the stubs:
```go
func verifyToken(cfg Config, header http.Header) error {
	got := header.Get(cfg.TokenHdr)
	if got == "" {
		return ErrMissingSignature
	}
	if subtle.ConstantTimeCompare([]byte(got), []byte(cfg.Token)) != 1 {
		return ErrBadSignature
	}
	return nil
}

func verifyBearer(cfg Config, header http.Header) error {
	raw := header.Get("Authorization")
	if raw == "" {
		return ErrMissingSignature
	}
	got := strings.TrimPrefix(raw, "Bearer ")
	if subtle.ConstantTimeCompare([]byte(got), []byte(cfg.Token)) != 1 {
		return ErrBadSignature
	}
	return nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/inboundauth/...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/inboundauth/verify.go internal/inboundauth/verify_test.go
git commit -m "feat(auth): token and bearer equality schemes"
```

---

## Task 6: Ed25519 scheme (Discord)

**Files:**
- Modify: `internal/inboundauth/verify.go` (replace `verifyEd25519` stub)
- Test: `internal/inboundauth/verify_test.go`

**Interfaces:**
- Produces: `verifyEd25519` verifying an `X-Signature-Ed25519` hex signature over `timestamp + rawBody` using hex-encoded `cfg.PublicKey`.

- [ ] **Step 1: Write the failing test**

Append to `internal/inboundauth/verify_test.go`:
```go
import "crypto/ed25519"

func TestEd25519Scheme(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}
	ts := "1700000000"
	body := []byte(`{"type":1}`)
	sig := ed25519.Sign(priv, append([]byte(ts), body...))

	cfg := Config{
		Scheme: SchemeEd25519, SigHeader: "X-Signature-Ed25519",
		TSHeader: "X-Signature-Timestamp", PublicKey: hex.EncodeToString(pub),
	}
	h := http.Header{}
	h.Set("X-Signature-Ed25519", hex.EncodeToString(sig))
	h.Set("X-Signature-Timestamp", ts)
	if err := Verify(cfg, h, body, time.Unix(0, 0)); err != nil {
		t.Fatalf("valid ed25519 sig rejected: %v", err)
	}
	// Tamper.
	if err := Verify(cfg, h, []byte("tampered"), time.Unix(0, 0)); err != ErrBadSignature {
		t.Fatalf("expected ErrBadSignature, got %v", err)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test -run TestEd25519 ./internal/inboundauth/...`
Expected: FAIL.

- [ ] **Step 3: Implement ed25519 verification**

Add `"crypto/ed25519"` to imports and replace the stub in `internal/inboundauth/verify.go`:
```go
func verifyEd25519(cfg Config, header http.Header, rawBody []byte, now time.Time) error {
	sigHex := header.Get(cfg.SigHeader)
	if sigHex == "" {
		return ErrMissingSignature
	}
	ts := header.Get(cfg.TSHeader)
	if ts == "" {
		return ErrMissingTimestamp
	}
	if cfg.TSToleranceS > 0 {
		if err := checkTolerance(ts, cfg.TSToleranceS, now); err != nil {
			return err
		}
	}
	pub, err := hex.DecodeString(cfg.PublicKey)
	if err != nil || len(pub) != ed25519.PublicKeySize {
		return ErrBadSignature
	}
	sig, err := hex.DecodeString(sigHex)
	if err != nil {
		return ErrBadSignature
	}
	msg := append([]byte(ts), rawBody...)
	if !ed25519.Verify(ed25519.PublicKey(pub), msg, sig) {
		return ErrBadSignature
	}
	return nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/inboundauth/...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/inboundauth/verify.go internal/inboundauth/verify_test.go
git commit -m "feat(auth): ed25519 verification scheme"
```

---

## Task 7: Preset library

**Files:**
- Create: `internal/inboundauth/presets.go`
- Test: `internal/inboundauth/presets_test.go`

**Interfaces:**
- Produces:
  - `func Preset(name string) (Config, bool)` — returns a base `Config` (axes filled, secret empty) for a preset name.
  - `func PresetNames() []Preset` where `type Preset struct { Name, Label string; NeedsSecret, NeedsPublicKey bool }` — drives the UI dropdown.

- [ ] **Step 1: Write the failing tests**

Create `internal/inboundauth/presets_test.go`:
```go
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
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test -run TestPreset ./internal/inboundauth/...`
Expected: FAIL (undefined `Preset`/`PresetNames`).

- [ ] **Step 3: Implement the presets**

Create `internal/inboundauth/presets.go`:
```go
package inboundauth

// Preset describes a selectable provider preset for the UI dropdown.
type Preset struct {
	Name           string
	Label          string
	NeedsSecret    bool
	NeedsPublicKey bool
}

// presetConfigs holds the axis values for each preset (secrets filled in by the user).
var presetConfigs = map[string]Config{
	"github": {
		Scheme: SchemeHMAC, Preset: "github", Algo: "sha256", Encoding: "hex",
		SigHeader: "X-Hub-Signature-256", SigParser: "plain", SigPrefix: "sha256=", Template: "raw_body",
	},
	"forgejo": {
		Scheme: SchemeHMAC, Preset: "forgejo", Algo: "sha256", Encoding: "hex",
		SigHeader: "X-Forgejo-Signature", SigParser: "plain", Template: "raw_body",
	},
	"stripe": {
		Scheme: SchemeHMAC, Preset: "stripe", Algo: "sha256", Encoding: "hex",
		SigHeader: "Stripe-Signature", SigParser: "kv-comma", Template: "ts.body", TSToleranceS: 300,
	},
	"slack": {
		Scheme: SchemeHMAC, Preset: "slack", Algo: "sha256", Encoding: "hex",
		SigHeader: "X-Slack-Signature", SigParser: "slack", Template: "slack_v0",
		TSHeader: "X-Slack-Request-Timestamp", TSToleranceS: 300,
	},
	"shopify": {
		Scheme: SchemeHMAC, Preset: "shopify", Algo: "sha256", Encoding: "base64",
		SigHeader: "X-Shopify-Hmac-Sha256", SigParser: "plain", Template: "raw_body",
	},
	"standard-webhooks": {
		Scheme: SchemeHMAC, Preset: "standard-webhooks", Algo: "sha256", Encoding: "base64",
		SigHeader: "webhook-signature", SigParser: "space-list", Template: "id.ts.body",
		TSHeader: "webhook-timestamp", IDHeader: "webhook-id", TSToleranceS: 300,
	},
	"gitlab-token": {
		Scheme: SchemeToken, Preset: "gitlab-token", TokenHdr: "X-Gitlab-Token",
	},
	"discord-ed25519": {
		Scheme: SchemeEd25519, Preset: "discord-ed25519",
		SigHeader: "X-Signature-Ed25519", TSHeader: "X-Signature-Timestamp",
	},
}

var presetMeta = []Preset{
	{"github", "GitHub", true, false},
	{"forgejo", "Forgejo / Gitea", true, false},
	{"stripe", "Stripe", true, false},
	{"slack", "Slack", true, false},
	{"shopify", "Shopify", true, false},
	{"standard-webhooks", "Svix / Standard Webhooks", true, false},
	{"gitlab-token", "GitLab (token)", true, false},
	{"discord-ed25519", "Discord (Ed25519)", false, true},
}

// Preset returns a base Config for name (secrets left empty), or false if unknown.
func Preset(name string) (Config, bool) {
	cfg, ok := presetConfigs[name]
	return cfg, ok
}

// PresetNames returns UI metadata for all presets in display order.
func PresetNames() []Preset {
	out := make([]Preset, len(presetMeta))
	copy(out, presetMeta)
	return out
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/inboundauth/...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/inboundauth/presets.go internal/inboundauth/presets_test.go
git commit -m "feat(auth): provider preset library"
```

---

## Task 8: Enforce verification in the ingest handler

**Files:**
- Modify: `internal/metrics/metrics.go` (add counter)
- Modify: `internal/handler/webhook.go:29-100` (verification gate)
- Test: `internal/handler/webhook_auth_integration_test.go`

**Interfaces:**
- Consumes: `inboundauth.ParseConfig`, `inboundauth.Verify`, `inboundauth.FailureReason`; `model.Source.AuthConfig`.
- Produces: `metrics.WebhookAuthFailures *prometheus.CounterVec` with labels `{source, reason}`.

- [ ] **Step 1: Add the metric**

Append to the `var (...)` block in `internal/metrics/metrics.go`:
```go
	WebhookAuthFailures = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "nitrohook_webhook_auth_failures_total",
		Help: "Total number of rejected webhooks due to failed authentication",
	}, []string{"source", "reason"})
```

- [ ] **Step 2: Write the failing integration test**

Create `internal/handler/webhook_auth_integration_test.go`:
```go
//go:build integration

package handler_test

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/zachbroad/nitrohook/internal/testutil"
)

func setAuth(t *testing.T, r *gin.Engine, slug string, cfg map[string]any) {
	t.Helper()
	s, _ := testutil.SetupTestDB(t) // returns the same shared store within a test run
	_ = s
	raw, _ := json.Marshal(cfg)
	// Persist via the store directly (no public API route for auth config).
	store := testAuthStore(t)
	if _, err := store.Sources.SetAuthConfig(context.Background(), slug, raw); err != nil {
		t.Fatalf("set auth config: %v", err)
	}
}
```
> **Note for implementer:** the existing `setupRouter` in `handler_integration_test.go` discards the `*store.Store`. Before writing this test, refactor `setupRouter` to also return the `*store.Store` (update its one caller), then use that store handle here instead of the `testAuthStore` placeholder above. Replace the body of `setAuth` with a call that uses the returned store. Keep the change minimal.

The actual test:
```go
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
```
> **Implementer note:** add a small `countDeliveries(t, store, slug)` helper using `store.Deliveries.List(ctx, &slug, 100)` and returning `len`. Add `"context"` to imports. Drop the placeholder `setAuth`/`testAuthStore` scaffolding — they were only illustrative; the real test writes config via the returned `store` directly.

- [ ] **Step 3: Run test to verify it fails**

Run: `go test -tags=integration -run TestIngestRejectsBadSignature ./internal/handler/...`
Expected: FAIL (no verification yet — bad request currently returns 202 and stores a delivery).

- [ ] **Step 4: Implement the verification gate**

In `internal/handler/webhook.go`, add imports `"github.com/zachbroad/nitrohook/internal/inboundauth"` and `"time"` (already present). Insert this block **after** the `json.Valid` check (line 48) and **before** the header-extraction comment (line 50):
```go
	// Authenticate the request against the source's configured scheme.
	authCfg, err := inboundauth.ParseConfig(src.AuthConfig)
	if err != nil {
		slog.Error("invalid source auth config", "error", err, "slug", sourceSlug)
		c.String(http.StatusInternalServerError, "invalid auth configuration")
		return
	}
	if err := inboundauth.Verify(authCfg, c.Request.Header, body, time.Now()); err != nil {
		metrics.WebhookAuthFailures.WithLabelValues(sourceSlug, inboundauth.FailureReason(err)).Inc()
		slog.Warn("webhook authentication failed", "slug", sourceSlug, "reason", inboundauth.FailureReason(err))
		c.String(http.StatusUnauthorized, "unauthorized")
		return
	}
```

- [ ] **Step 5: Run tests to verify they pass**

Run:
```bash
go test -tags=integration -run 'TestIngest' ./internal/handler/...
go build ./...
```
Expected: PASS; existing ingest tests (no auth config) still pass, proving backward compatibility.

- [ ] **Step 6: Commit**

```bash
git add internal/metrics/metrics.go internal/handler/webhook.go \
        internal/handler/handler_integration_test.go internal/handler/webhook_auth_integration_test.go
git commit -m "feat(auth): verify incoming webhooks before persistence"
```

---

## Task 9: Web UI — auth-card fragment and handler

**Files:**
- Modify: `web/source.go` (add `UpdateSourceAuth`; extend `sourceData`; render auth-card in `SourceDetail`)
- Modify: `web/handler.go:139-152` (`sourceData` fields)
- Modify: `web/templates/source-overview.html` (render + define `auth-card`)
- Modify: `cmd/api/main.go` (route)
- Test: `internal/handler`-style is not applicable; use a build + manual smoke, plus an integration test of the handler writing config.

**Interfaces:**
- Consumes: `store.SetAuthConfig`, `inboundauth.Preset`, `inboundauth.PresetNames`.
- Produces: `POST /sources/:slug/auth` → re-rendered `auth-card` fragment; `sourceData.AuthPresets []inboundauth.Preset`, `sourceData.AuthConfig inboundauth.Config`, `sourceData.AuthError`, `sourceData.AuthSuccess`.

- [ ] **Step 1: Extend `sourceData`**

In `web/handler.go`, add to the `sourceData` struct and import `inboundauth`:
```go
	AuthPresets   []inboundauth.Preset
	AuthConfig    inboundauth.Config
	AuthEnabled   bool
	AuthError     string
	AuthSuccess   string
```
Add import: `"github.com/zachbroad/nitrohook/internal/inboundauth"`.

- [ ] **Step 2: Populate auth data in `SourceDetail`**

In `web/source.go`, replace the `SourceDetail` render call (lines 66-70) to decode the stored config and pass presets:
```go
	authCfg, _ := inboundauth.ParseConfig(source.AuthConfig)
	h.render(c, "source-overview", sourceData{
		Nav:         "sources",
		Source:      source,
		WebhookURL:  webhookURL(c, source.Slug),
		AuthPresets: inboundauth.PresetNames(),
		AuthConfig:  authCfg,
		AuthEnabled: authCfg.Scheme != inboundauth.SchemeNone,
	})
```
Add import `"github.com/zachbroad/nitrohook/internal/inboundauth"` to `web/source.go`.

- [ ] **Step 3: Add the `UpdateSourceAuth` handler**

Append to `web/source.go`:
```go
// UpdateSourceAuth saves a source's incoming-webhook auth config from the auth-card form.
func (h *Handler) UpdateSourceAuth(c *gin.Context) {
	slug := c.Param("slug")
	source, err := h.store.Sources.GetBySlug(c.Request.Context(), slug)
	if err != nil {
		c.String(http.StatusNotFound, "Source not found")
		return
	}

	enabled := c.PostForm("enabled") == "on" || c.PostForm("enabled") == "true"
	var authErr, authOK string
	var cfg inboundauth.Config

	if !enabled {
		if _, err := h.store.Sources.SetAuthConfig(c.Request.Context(), slug, nil); err != nil {
			authErr = "Failed to disable authentication"
		} else {
			authOK = "Authentication disabled"
		}
	} else {
		presetName := c.PostForm("preset")
		base, ok := inboundauth.Preset(presetName)
		if !ok {
			authErr = "Unknown preset"
		} else {
			base.Secret = strings.TrimSpace(c.PostForm("secret"))
			base.Token = strings.TrimSpace(c.PostForm("secret")) // token/bearer reuse the secret field
			base.PublicKey = strings.TrimSpace(c.PostForm("public_key"))
			raw, _ := json.Marshal(base)
			if _, err := h.store.Sources.SetAuthConfig(c.Request.Context(), slug, raw); err != nil {
				authErr = "Failed to save authentication"
			} else {
				authOK = "Authentication saved"
				cfg = base
			}
		}
	}

	source, _ = h.store.Sources.GetBySlug(c.Request.Context(), slug)
	if cfg.Scheme == "" {
		cfg, _ = inboundauth.ParseConfig(source.AuthConfig)
	}
	h.renderFragment(c, "source-overview", "auth-card", sourceData{
		Source:      source,
		AuthPresets: inboundauth.PresetNames(),
		AuthConfig:  cfg,
		AuthEnabled: cfg.Scheme != inboundauth.SchemeNone,
		AuthError:   authErr,
		AuthSuccess: authOK,
	})
}
```
(`encoding/json` and `strings` are already imported in `web/source.go`.)

- [ ] **Step 4: Add the auth-card template**

In `web/templates/source-overview.html`, add `{{template "auth-card" .}}` on the line after `{{template "mode-card" .}}` (inside the `content` block), and append this fragment at the end of the file:
```html
{{define "auth-card"}}
<div class="card" id="auth-card">
  <h2>Authentication</h2>
  <p style="font-size:0.85rem;color:var(--text-muted);margin-bottom:0.75rem">
    Verify incoming webhooks before they are stored. Choose a provider preset and paste the shared secret from the sending platform.
  </p>
  {{if .AuthError}}<p style="color:var(--danger);font-size:0.85rem">{{.AuthError}}</p>{{end}}
  {{if .AuthSuccess}}<p style="color:var(--success);font-size:0.85rem">{{.AuthSuccess}}</p>{{end}}
  <form hx-post="/sources/{{.Source.Slug}}/auth" hx-target="#auth-card" hx-swap="outerHTML">
    <label style="display:flex;align-items:center;gap:0.5rem;margin-bottom:0.75rem">
      <input type="checkbox" name="enabled" {{if .AuthEnabled}}checked{{end}}>
      Require authentication
    </label>
    <div class="form-row" style="margin-bottom:0.75rem">
      <label>Provider</label>
      <select name="preset">
        {{range .AuthPresets}}
        <option value="{{.Name}}" {{if eq $.AuthConfig.Preset .Name}}selected{{end}}>{{.Label}}</option>
        {{end}}
      </select>
    </div>
    <div class="form-row" style="margin-bottom:0.75rem">
      <label>Shared secret / token</label>
      <input type="password" name="secret" placeholder="Paste the signing secret" autocomplete="off">
    </div>
    <div class="form-row" style="margin-bottom:0.75rem">
      <label>Public key (Ed25519 only)</label>
      <input type="text" name="public_key" value="{{.AuthConfig.PublicKey}}" placeholder="Hex-encoded Ed25519 public key" autocomplete="off">
    </div>
    <p style="font-size:0.8rem;color:var(--text-muted);margin-bottom:0.75rem">
      HMAC and Ed25519 verify a signature without exposing the secret on the wire. Bearer/token schemes send the secret with every request — prefer a signature scheme when the sender supports one.
    </p>
    <button type="submit" class="btn btn-primary btn-sm">Save Authentication</button>
  </form>
</div>
{{end}}
```
> **Implementer note:** confirm the CSS classes `form-row`, `--danger`, `--success` exist in the stylesheet; if not, reuse classes already present in `source-overview.html`/`layout.html` (grep the templates) rather than inventing new ones.

- [ ] **Step 5: Register the route**

In `cmd/api/main.go`, after the `r.POST("/sources/:slug/mode", webH.UpdateSourceMode)` line, add:
```go
	r.POST("/sources/:slug/auth", webH.UpdateSourceAuth)
```

- [ ] **Step 6: Build, then smoke-test manually**

Run:
```bash
go build ./...
make run-api
```
Then in a browser: open a source, enable authentication, pick "GitHub", paste a secret, Save — confirm the auth-card re-renders with "Authentication saved". Send a signed and an unsigned webhook with `curl` and confirm `202` vs `401`. Verify the config persisted:
```bash
curl -s localhost:8080/api/sources/<slug> | grep auth_config
```

- [ ] **Step 7: Commit**

```bash
git add web/source.go web/handler.go web/templates/source-overview.html cmd/api/main.go
git commit -m "feat(auth): source auth-config UI"
```

---

## Task 10: Documentation

**Files:**
- Create: `docs/src/content/docs/guides/authentication.md`
- Modify: `CLAUDE.md` (add `inboundauth` to the package table)

**Interfaces:** none (docs only).

- [ ] **Step 1: Write the guide**

Create `docs/src/content/docs/guides/authentication.md` with frontmatter matching the existing guides (check an existing file like `docs/src/content/docs/guides/scripting.md` for the exact frontmatter shape), covering: enabling auth on a source, the preset list and which header/secret each expects, the security note on signature vs bearer schemes, the timestamp-tolerance behavior for Stripe/Slack/Svix, and that a `401` is returned (and `nitrohook_webhook_auth_failures_total` incremented) on failure.

- [ ] **Step 2: Add the package to CLAUDE.md**

In `CLAUDE.md`, add a row to the "Key internal packages" table:
```markdown
| `inboundauth` | Verifies incoming webhook authenticity (HMAC/bearer/token/Ed25519) against per-source config, with provider presets |
```

- [ ] **Step 3: Commit**

```bash
git add docs/src/content/docs/guides/authentication.md CLAUDE.md
git commit -m "docs(auth): incoming webhook authentication guide"
```

---

## Self-Review

**Spec coverage:**
- Schemes none/hmac/bearer/token/ed25519 → Tasks 2, 3-4, 5, 6. ✓
- Presets (github, forgejo, stripe, slack, shopify, standard-webhooks, gitlab-token, discord-ed25519, custom) → Task 7; "custom" is any preset with hand-edited axes / the base config path. ✓
- Timestamp/replay only where preset requires → `ts.body`/`slack_v0`/`id.ts.body` templates carry `TSToleranceS`; raw-body presets have none. ✓
- Storage `auth_config JSONB`, plaintext secret → Task 1. ✓
- Verify before persist/publish, 401, no storage → Task 8 (test asserts 0 deliveries). ✓
- Prometheus counter `{source, reason}` → Task 8. ✓
- Backward compatibility → Task 1 (nil config) + Task 8 (existing ingest tests still pass). ✓
- Custom-mode boundary (presets cover exotic parsers) → parsers live in Task 4; custom exposes simple-HMAC via editable axes. ✓
- Constant-time comparison → `subtle.ConstantTimeCompare`/`hmac`/`ed25519.Verify` throughout. ✓
- Twilio/Basic excluded → not implemented; noted in Global Constraints. ✓

**Type consistency:** `Config` field names are identical across config.go, verify.go, presets.go, and the handler JSON. `SetAuthConfig`, `ParseConfig`, `Verify`, `FailureReason`, `Preset`, `PresetNames` are referenced with consistent signatures. `extractSignatures` (plural, `[]string`) replaces the Task 3 singular `extractSignature` — Task 4 explicitly deletes the old function to avoid a dangling reference.

**Placeholder scan:** Task 8's test contains illustrative scaffolding (`setAuth`/`testAuthStore`) that the implementer notes instruct to remove in favor of the store handle returned by the refactored `setupRouter`; this is called out explicitly, not left as a silent TODO.

**Known deviation from spec (flag to owner):** The spec proposed refactoring `store.Update` to an options struct. This plan instead adds a dedicated `SetAuthConfig` method and leaves `Update` untouched, to minimize blast radius in a security-sensitive PR (the `Update` refactor would ripple to ~6 call sites). Functionally equivalent for the feature; the broader refactor can be a separate cleanup issue.
