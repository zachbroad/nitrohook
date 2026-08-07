# Preset/Custom auth split, basic auth, and secret generation for inbound webhook auth

Date: 2026-07-15
Context: GitHub issue #1 — Eli asked for bearer + basic auth support and a
copy-paste token-generation flow. HMAC/bearer/token/ed25519 verifiers exist,
but the API/UI only accept provider presets: bearer is unreachable, basic is
absent, and there is no way to generate a secret for user-minted-secret
presets (GitHub, Forgejo, GitLab).

## Goals

1. Split auth configuration into two trees: **Preset** (pick a provider) and
   **Custom** (pick a scheme manually).
2. Custom tree offers: **Bearer**, **Basic** (new scheme, RFC 7617,
   bcrypt-hashed password), **Token** (custom header name), and **HMAC**
   (custom signature header, sha256/sha1, hex/base64, optional prefix,
   raw-body template).
3. Client-side "Generate" button wherever the user mints the secret: the
   user-minted presets (GitHub, Forgejo, GitLab) and all custom schemes.

Non-goals: custom timestamp templates / signature parsers / Ed25519 in the
Custom tree (preset-only), secret rotation, idempotency/replay dedup (issue
#9), encrypting other secrets at rest.

## Data model

Stored `sources.auth_config` keeps the existing `inboundauth.Config` shape.
Mode is derived, not stored separately: `Preset != ""` means preset mode;
`Preset == ""` with a scheme set means custom mode. Custom HMAC configs
always use `Template: "raw_body"` and `SigParser: "plain"`.

## Backend

### `internal/inboundauth/config.go`

- Add `SchemeBasic Scheme = "basic"`.
- Add `Config` fields: `Username string` (`json:"username,omitempty"`,
  plaintext) and `PasswordHash string` (`json:"password_hash,omitempty"`,
  bcrypt hash).

### `internal/inboundauth/verify.go`

New `verifyBasic(cfg, header)`:

- Fail closed with `ErrMissingSecret` if `Username` or `PasswordHash` is empty.
- Missing `Authorization` header → `ErrMissingSignature`.
- Scheme keyword `Basic ` matched case-insensitively (same approach as
  `verifyBearer`); anything else → `ErrBadSignature`.
- Base64-decode the credentials; split on the first `:`. Malformed input →
  `ErrBadSignature`.
- Username compared with `subtle.ConstantTimeCompare`; password checked with
  `bcrypt.CompareHashAndPassword`. Either mismatch → `ErrBadSignature`.
- Wire `SchemeBasic` into the `Verify` switch.

### `internal/inboundauth/presets.go`

- No new presets (bearer/basic/token/custom-HMAC live in the Custom tree).
- `PresetInfo` gains `SecretSource string` (`json:"secret_source"`):
  `"platform"` (copy from the sender's dashboard — Stripe, Slack, Shopify,
  Svix, Discord) or `"user"` (user mints the secret — GitHub, Forgejo,
  GitLab). Drives the Generate button in the UI.

### `internal/handler/source_auth.go`

`updateSourceAuthRequest` becomes mode-aware:

```json
{
  "enabled": true,
  "mode": "preset" | "custom",

  "preset": "github",              // preset mode only

  "scheme": "hmac|bearer|basic|token",  // custom mode only
  "sig_header": "X-Foo-Signature",      // custom hmac
  "algo": "sha256" | "sha1",            // custom hmac (default sha256)
  "encoding": "hex" | "base64",         // custom hmac (default hex)
  "sig_prefix": "sha256=",              // custom hmac, optional
  "token_header": "X-Foo-Token",        // custom token

  "secret": "...",                 // hmac/bearer/token credential
  "username": "...", "password": "...", // basic only
  "public_key": "..."              // preset mode (Discord) only
}
```

Validation:

- `mode: "preset"` — unchanged behavior: known preset required, plus the
  credential the preset needs. Unknown fields for the mode are ignored.
- `mode: "custom"` — scheme must be one of hmac/bearer/basic/token.
  - hmac: `sig_header` and `secret` required; algo/encoding validated
    against the allowed values, defaulted when empty; `sig_prefix` optional.
  - bearer: `secret` required (stored in `Config.Token`).
  - token: `token_header` and `secret` required (stored in `Config.Token`).
  - basic: `username` and `password` required; password hashed with
    `bcrypt.GenerateFromPassword` at **cost 6** (verification runs on the
    unauthenticated webhook ingest hot path; cost 6 is ~3ms vs ~60ms at the
    default cost 10) and only the hash stored in `Config.PasswordHash`.
- Missing/invalid anything → 400 with a specific message.
- Requests without `mode` default to `"preset"` (backward compatible).

### `internal/inboundauth/sanitize.go`

`Sanitized` gains the non-secret axes the UI needs to redisplay a saved
config:

- `Username string` (`json:"username,omitempty"`)
- `SigHeader`, `Algo`, `Encoding`, `SigPrefix`, `TokenHeader` (all
  `omitempty`)
- `HasSecret` also true when `PasswordHash != ""`.

The password hash, secret, and token are never exposed.

## Frontend (`web/ui/src`)

### `lib/types.ts`

- `AuthPreset` gains `secret_source: "platform" | "user"`.
- `SourceAuthConfig` gains `username`, `sig_header`, `algo`, `encoding`,
  `sig_prefix`, `token_header` (all optional).

### `lib/api.ts` / `lib/queries.ts`

- `updateSourceAuth` payload gains `mode`, `scheme`, and the custom-mode
  fields above.

### `routes/source-auth-form.tsx`

Two trees under the existing enable switch, selected by a **Preset / Custom**
segmented control (initialized from the saved config's derived mode):

- **Preset tree** — existing UI: provider dropdown, secret or public-key
  input. Generate button shown when `secret_source === "user"`.
- **Custom tree** — scheme select (HMAC signature, Bearer token, Basic auth,
  Token header), then scheme-specific fields:
  - HMAC: signature header, algorithm (sha256/sha1), encoding (hex/base64),
    optional signature prefix, secret.
  - Bearer: secret only.
  - Token: header name + secret.
  - Basic: username + password.
  - Generate button always shown on the secret/password field (all custom
    schemes are user-minted).

Generate button behavior (both trees):

- Fills the field with `hex(crypto.getRandomValues(new Uint8Array(32)))`,
  switches the input from `type="password"` to visible text, and shows a
  **Copy** button (clipboard API) — the one chance to copy the value out,
  since saved secrets are never echoed back. The field stays visible once
  generated; a successful save clears it.

Validation mirrors the server rules; the existing "re-enter the secret to
save changes" rule applies to the password and all custom secrets too. The
existing HMAC-preference security note stays visible in both trees.

## Docs & changelog

- `docs/src/content/docs/guides/authentication.md`: document the
  Preset/Custom split; add a "Custom schemes" section (HMAC, Bearer, Basic,
  Token — the non-signature ones marked "secret travels on the wire"); note
  that the Basic password is stored as a bcrypt hash (exception to the
  plaintext-at-rest note); document the Generate button.
- `docs/src/content/docs/api/sources` (update-source-authentication section):
  document the new request shape.
- `CHANGELOG.md` under `[Unreleased]` / `Added`: custom auth schemes
  (HMAC/bearer/basic/token), basic auth, secret generation in the UI.

## Testing

- `verify_test.go`: table-driven `basic` cases — valid creds; wrong password;
  wrong username; missing header; non-Basic scheme; malformed base64; decoded
  value without `:`; case-insensitive `bAsIc` prefix; empty username/hash
  fails closed.
- `presets_test.go`: `SecretSource` metadata correct for all presets.
- `source_auth_test.go`:
  - preset mode unchanged (no `mode` field defaults to preset).
  - custom hmac: missing sig_header/secret → 400; bad algo/encoding → 400;
    defaults applied; stored config has raw_body/plain.
  - custom bearer/token: required fields enforced; secret stored in Token.
  - basic: missing username or password → 400; stored config contains a
    bcrypt hash, not the plaintext password.
  - sanitized responses carry the non-secret axes and `username`, never
    secret/token/hash material.
- Handler integration test (`webhook_auth_integration_test.go`): end-to-end
  401/2xx for custom bearer, basic, and custom HMAC against a live source.

## Error handling

- All new failure paths map onto existing sentinel errors, so the Prometheus
  `reason` labels (`missing_signature`, `bad_signature`, `missing_secret`)
  work unchanged.
- Corrupt configs still sanitize to disabled (existing behavior).
