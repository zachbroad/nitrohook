# Changelog

All notable changes to NitroHook are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- Incoming webhook authentication: verify each webhook against per-source config before it is stored or fanned out. Supports HMAC-SHA256/SHA1 signatures, bearer/plain-token, and Ed25519, with presets for GitHub, Forgejo, Stripe, Slack, Shopify, Svix/Standard Webhooks, GitLab, and Discord. Configurable per source in the UI.
- Web UI: search filter in the sources and deliveries sidebars, filtering the list by name/slug or ID as you type. The search term is kept in the URL (`?q=`) so it survives a refresh.
- Web UI: clicking a delivery row on a source's Events tab now opens the delivery detail (headers, payload, attempts) in a modal instead of navigating away. The open delivery is kept in the URL (`?delivery=`) so it survives a refresh.
- Web UI: clicking an action row on a source's Actions tab now opens the edit form in a modal, matching the events tab. The open action is kept in the URL (`?action=`) so it survives a refresh.
- Web UI: checkbox multi-select on a source's Events tab — pick individual recorded deliveries (or select all via the header checkbox) and forward them in one go with "Forward selected", alongside the existing "Forward all". The per-row Forward button and its Actions column were removed in favor of selection.
- Web UI: the deliveries status filter is kept in the URL (`?status=`) so filtered views survive a refresh and can be shared; the dashboard's Failed tile and status legend deep-link to the matching filtered list.
- REST API endpoints for incoming webhook authentication: `GET /api/auth/presets` lists the provider presets, and `PUT /api/sources/{slug}/auth` enables, updates, or disables a source's auth config.
- Web UI: Authentication card on a source's Overview tab — toggle verification on, pick a provider preset, and paste the shared secret or Ed25519 public key.
- Web UI: toast confirmations for write actions — creating, updating, or deleting a source or action, saving a transform script, toggling a source's mode, and forwarding deliveries now show a success toast (failures already surfaced an error toast).

### Security

- Webhook sources can now require authentication; unauthenticated requests are rejected with HTTP 401 before any delivery is stored or published.
- API responses never expose stored webhook credentials: source endpoints return a sanitized `auth_config` (`enabled`, `scheme`, `preset`, `public_key`, `has_secret`) instead of the raw config, and the MCP server's `list_sources` omits `auth_config` entirely.

### Fixed

- Web UI now shows inline error panels with a Retry button when data fails to load, instead of misleading empty states ("No sources yet", "Delivery not found") when the API is unreachable; 4xx responses are no longer retried before surfacing.

## [0.1.0] - 2026-07-10

### Added

- `make dev` — one-command local development loop: starts Postgres and Redis in
  Docker (waiting until healthy), applies migrations via the API binary, then runs
  the API with an in-process worker under [Air](https://github.com/air-verse/air)
  for hot reload.
- `make dev-setup` — installs the Air hot-reload tool.
- `make dev-down` — stops the Postgres and Redis containers started by `make dev`.

[Unreleased]: https://github.com/zachbroad/nitrohook/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/zachbroad/nitrohook/releases/tag/v0.1.0
