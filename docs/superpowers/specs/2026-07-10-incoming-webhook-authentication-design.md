# Incoming Webhook Authentication — Design

**Issue:** [#1 Authentication](https://github.com/zachbroad/nitrohook/issues/1)
**Date:** 2026-07-10
**Status:** Approved, ready for implementation plan

## Problem

NitroHook ingests webhooks at `POST /webhooks/:sourceSlug` with **no authentication** — any
party who knows (or guesses) a source slug can inject payloads that get stored, published to
Redis, and fanned out to actions. The primary driver is a self-hosted Forgejo → NitroHook
deployment, but the feature should verify webhooks from any major platform.

## Goals

- Verify incoming webhooks against per-source auth config **before** anything is persisted or
  published; reject unauthenticated requests with `401`.
- Support the signature schemes used by the major platforms, driven by a **preset library** over
  a **parameterized verifier** — not bespoke per-platform code.
- Remain fully backward-compatible: sources with no auth config behave exactly as today.

## Non-goals (deferred to follow-up issues)

- Secret rotation (accept old + new during a window).
- HTTPS-only enforcement (reject plain-HTTP endpoints).
- Encryption of secrets at rest.
- Recording/persisting failed auth attempts (only a metric counter in v1).
- **Twilio** preset — its signature covers the URL + alphabetically-sorted POST params rather
  than the raw body, a distinct model that doesn't fit the parameterized-verifier axes. Left out
  of the v1 preset library.
- Basic auth (Eli's proposal) — superseded by HMAC-first + Bearer.

## Background: how major platforms sign (2026 research)

| Platform | Header | Algo | Enc | Prefix | Signs | Timestamp |
|---|---|---|---|---|---|---|
| GitHub | `X-Hub-Signature-256` | HMAC-SHA256 (+SHA1 legacy) | hex | `sha256=` | raw body | no |
| Forgejo/Gitea | `X-Forgejo-Signature` (+GitHub-compat) | HMAC-SHA256 | hex | none (native) | raw body | no |
| GitLab | `X-Gitlab-Token` (default) | **none — plain token** | — | — | nothing | no |
| Stripe | `Stripe-Signature` | HMAC-SHA256 | hex | `t=`,`v1=` | `{ts}.{body}` | yes (5m) |
| Shopify | `X-Shopify-Hmac-Sha256` | HMAC-SHA256 | base64 | none | raw body | no |
| Slack | `X-Slack-Signature` | HMAC-SHA256 | hex | `v0=` | `v0:{ts}:{body}` | yes (5m) |
| Twilio | `X-Twilio-Signature` | HMAC-**SHA1** | base64 | none | URL + sorted params | no |
| Svix / Standard Webhooks | `webhook-signature` | HMAC-SHA256 (+Ed25519 opt) | base64 | `v1,` | `{id}.{ts}.{body}` | yes |
| Discord | `X-Signature-Ed25519` | **Ed25519** (asymmetric) | hex | none | `ts`+body | yes |

**Verdict:** HMAC-SHA256 is the norm (6/9 primary), but four axes vary independently in the wild:
algorithm (SHA256/SHA1), encoding (hex/base64, ~50/50 split), header name + prefix parsing, and
the signed-string template (raw body vs `{ts}.{body}` vs `{id}.{ts}.{body}` vs `v0:{ts}:{body}`).
Two platforms sit outside HMAC: GitLab (plain-token equality) and Discord (Ed25519).

## Design decisions (settled)

1. **Schemes in v1:** `none` (default), `hmac`, `bearer`, `token` (plain equality), `ed25519`.
2. **Preset library + parameterized verifier** (not bespoke per-platform verifiers, not
   custom-only).
3. **Timestamp/replay** is included only where a preset intrinsically requires it (Stripe, Slack,
   Svix); it is not a standalone toggle.
4. **Storage:** `auth_config JSONB` column on `sources`; secret/token/public-key stored plaintext
   inside it (matches `actions.signing_secret`). Encryption deferred.
5. **Prometheus counter** `webhook_auth_failures_total{source}` on every rejection (firm
   requirement, not optional).
6. **Custom-mode boundary:** presets cover the exotic multi-field parsers (Stripe `kv-comma`,
   Svix `space-list`, Slack `v0=`); Custom mode exposes only the "simple HMAC family"
   (configurable header, algo, encoding, optional prefix strip, raw-body or `ts.body`).

## Architecture

### Data model & storage

- **Migration `000009_add_source_auth`** — add nullable `auth_config JSONB` to `sources`.
  `NULL`/absent ⇒ scheme `none` ⇒ backward-compatible. Down migration drops the column.
- `model.Source` gains `AuthConfig json.RawMessage` (mirrors `Action.Config`).
- `store/source.go` — extend all SELECT column lists to include `auth_config`; extend
  `Create`/`Update`. **Targeted cleanup:** refactor `Update`'s positional-nil-pointer signature
  toward an options struct, since `auth_config` is the third optional field and the pattern is
  already unwieldy.

### New package `internal/inboundauth`

Isolated, independently testable verification logic.

- `Config` — decodes the JSONB. Fields: `scheme`, `preset`, and per-scheme:
  `algo` (sha256|sha1), `encoding` (hex|base64), `sig_header`, `sig_prefix`, `template`,
  `ts_header`, `ts_tolerance_secs`, `secret`, `token`, `public_key`.
- `Verify(cfg Config, req *http.Request, rawBody []byte) error` — returns `nil` on success or a
  typed failure (used to drive the 401 and the metric label).
- **Header parsers** (the provider-specific complexity) modeled as a small named set:
  - `plain` — whole header is the signature, optional fixed prefix strip (`sha256=`, none).
  - `kv-comma` — Stripe: parse `t=`,`v1=` from comma-separated `k=v`.
  - `space-list` — Svix: space-separated `v1,<b64>` tokens.
  - `slack` — header is `v0=<hex>`, timestamp from a separate header.
- **Signed-string templates:** `raw_body`, `ts.body`, `id.ts.body`, `slack_v0` (`v0:{ts}:{body}`).
- **Presets** wire parser + template + axes + which fields the UI collects.
- **Ed25519** verifies signature over the platform's `ts`+body construction using stored
  `public_key`. **token** does constant-time equality on a header value. **bearer** does
  constant-time equality on `Authorization: Bearer <token>`.
- Constant-time comparison everywhere (reuse the `hmac.Equal`/`subtle` approach from `signing`).
  `signing.Verify` is unchanged and continues to serve *outgoing* signing.

### Ingestion integration (`handler/webhook.go`)

- Insert verification **after** `io.ReadAll` + `json.Valid` (raw bytes available) and **before**
  `Deliveries.Create`. `src` (with `auth_config`) is already loaded.
- On failure: increment `webhook_auth_failures_total{source}`, return `401` with a generic body,
  persist/publish nothing.
- `scheme: none` (or absent config): skip verification, behave exactly as today.

### Web UI (`web/source.go`, `web/templates/source-overview.html`, `cmd/api/main.go`)

- New `{{define "auth-card"}}` fragment on the source overview tab, copying the existing
  `mode-card` htmx pattern (`hx-post` → fragment re-render).
- New route `POST /sources/:slug/auth` → handler returns the re-rendered `auth-card` fragment via
  `renderFragment`.
- New fields on the `sourceData` struct for auth state.
- Flow: enable toggle → **preset dropdown** → paste secret → save. **Custom** reveals the axis
  fields. **Bearer** shows a security disclaimer (secret travels on the wire) and a
  "generate token" button.

## Testing

- Table-driven unit tests in `inboundauth` with **real captured signature vectors** per preset
  (known secret + body + expected signature for GitHub, Forgejo, Stripe, Slack, Shopify, Svix,
  Discord, GitLab). These double as executable documentation of each scheme.
- Negative cases: tampered body, wrong secret, expired/future timestamp, missing header,
  wrong scheme.
- Handler test: bad signature ⇒ `401`, **no delivery row created, no Redis publish**, counter
  incremented.
- Backward-compat: source with no `auth_config` ingests as before.

## Touch points summary

- `migrations/000009_add_source_auth.{up,down}.sql`
- `internal/model/model.go` — `Source.AuthConfig`
- `internal/store/source.go` — SELECTs, `Create`, `Update` (+ options-struct cleanup)
- `internal/inboundauth/` — new package (config, verifier, header parsers, presets, tests)
- `internal/handler/webhook.go` — verification gate + 401 + metric
- Prometheus metric registration (alongside existing monitoring setup)
- `web/source.go`, `web/templates/source-overview.html`, `web/handler.go` (`sourceData`),
  `cmd/api/main.go` (route)

## Collaboration

Issue #1 is a collaboration with contributor Eli (proposed Bearer/Basic; owner prefers
HMAC-first). This design centers HMAC while honoring the Bearer request (with disclaimer +
generate-token UX). Basic auth is intentionally out of scope for v1. No design summary will be
posted to the issue per owner instruction.
