# Basic auth, Bearer preset, and secret generation for inbound webhook auth

Date: 2026-07-15
Context: GitHub issue #1 — Eli asked for bearer + basic auth support and a
copy-paste token-generation flow. HMAC/bearer/token/ed25519 verifiers exist;
bearer is unreachable (no preset), basic is absent, and there is no way to
generate a secret for user-minted-secret presets (GitHub, Forgejo, GitLab).

## Goals

1. Add a `basic` auth scheme (RFC 7617) with bcrypt-hashed password storage.
2. Expose the existing bearer verifier via a `bearer` preset.
3. Add a client-side "Generate" button for presets where the user mints the
   secret, so they can create a strong secret and copy it into the sender.

Non-goals: secret rotation, idempotency/replay dedup (issue #9), encrypting
other secrets at rest.

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

- New presets:
  - `bearer` — `Scheme: SchemeBearer`, label "Bearer token". Verifies
    `Authorization: Bearer <token>` via the existing verifier.
  - `basic` — `Scheme: SchemeBasic`, label "Basic auth".
- `PresetInfo` gains:
  - `SecretSource string` (`json:"secret_source"`): `"platform"` (copy from
    the sender's dashboard — Stripe, Slack, Shopify, Svix, Discord) or
    `"user"` (user mints the secret — GitHub, Forgejo, GitLab, Bearer,
    Basic). Drives the Generate button in the UI.
  - `NeedsUsername bool` (`json:"needs_username"`): true only for `basic`.
- `basic` has `NeedsSecret: true` (the password field doubles as the secret
  input in the UI).

### `internal/handler/source_auth.go`

- `updateSourceAuthRequest` gains `Username` and `Password`.
- For the `basic` preset: require both username and password (400 otherwise);
  hash the password with `bcrypt.GenerateFromPassword` at **cost 6** (chosen
  below default cost 10 because verification runs on the unauthenticated
  webhook ingest hot path; cost 6 is ~3ms vs ~60ms) and store only the hash
  in `Config.PasswordHash`. Do not set `Secret`/`Token` for basic.
- Other presets ignore username/password.

### `internal/inboundauth/sanitize.go`

- `Sanitized` gains `Username string` (`json:"username,omitempty"`) so the UI
  can redisplay it. The password hash is never exposed.
- `HasSecret` also true when `PasswordHash != ""`.

## Frontend (`web/ui/src`)

### `lib/types.ts`

- `AuthPreset` gains `secret_source: "platform" | "user"` and
  `needs_username: boolean`.
- `SourceAuthConfig` gains `username?: string`.

### `lib/api.ts` / `lib/queries.ts`

- `updateSourceAuth` payload gains optional `username` and `password`.

### `routes/source-auth-form.tsx`

- When `presetInfo.needs_username`: show a Username input (plaintext,
  prefilled from `auth.username`) and relabel the secret field "Password".
- When `presetInfo.secret_source === "user"`: show a **Generate** button next
  to the secret/password input. Clicking it:
  - fills the field with `hex(crypto.getRandomValues(new Uint8Array(32)))`,
  - switches the input from `type="password"` to visible text,
  - shows a **Copy** button (clipboard API) — this is the one chance to copy
    the value out, since saved secrets are never echoed back.
  - Manual typing after generation reverts nothing; the field stays visible
    once generated until save clears it.
- Validation: basic requires non-empty username and password; the existing
  "re-enter the secret to save changes" rule applies to the password too.

## Docs & changelog

- `docs/src/content/docs/guides/authentication.md`: add `Bearer token` and
  `Basic auth` rows to the presets table (both marked "not a signature —
  secret travels on the wire"); note that the Basic password is stored as a
  bcrypt hash (exception to the plaintext-at-rest note); document the
  Generate button in the enabling steps.
- `CHANGELOG.md` under `[Unreleased]` / `Added`: bearer preset, basic auth
  scheme, secret generation in the UI.

## Testing

- `verify_test.go`: table-driven `basic` cases — valid creds; wrong password;
  wrong username; missing header; non-Basic scheme; malformed base64; decoded
  value without `:`; case-insensitive `bAsIc` prefix; empty
  username/hash fails closed. Bearer preset resolves and verifies end-to-end.
- `presets_test.go`: new presets present; `SecretSource`/`NeedsUsername`
  metadata correct for all presets.
- `source_auth_test.go`: basic without username or password → 400; stored
  config contains a bcrypt hash, not the plaintext password; sanitized
  response carries `username` and `has_secret: true` but no hash; bearer
  preset stores token and round-trips.
- Handler integration test (`webhook_auth_integration_test.go`): end-to-end
  401/2xx for basic and bearer against a live source.

## Error handling

- All new failure paths map onto existing sentinel errors, so the Prometheus
  `reason` labels (`missing_signature`, `bad_signature`, `missing_secret`)
  work unchanged for basic/bearer.
- Corrupt configs still sanitize to disabled (existing behavior).
