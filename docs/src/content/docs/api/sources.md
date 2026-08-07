---
title: Sources API
description: REST API reference for managing webhook sources.
---

Base path: `/api/sources`

## List sources

```http
GET /api/sources
```

Returns all sources.

## Create source

```http
POST /api/sources
Content-Type: application/json

{
  "name": "My App",
  "slug": "my-app"
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | Yes | Display name |
| `slug` | string | Yes | URL-safe identifier, used in webhook URL |

## Get source

```http
GET /api/sources/{slug}
```

## Update source

```http
PATCH /api/sources/{slug}
Content-Type: application/json

{
  "name": "Updated Name"
}
```

## Delete source

```http
DELETE /api/sources/{slug}
```

## Update source authentication

```http
PUT /api/sources/{slug}/auth
Content-Type: application/json

{
  "enabled": true,
  "preset": "github",
  "secret": "my-webhook-secret"
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `enabled` | boolean | Yes | `false` disables verification and clears the stored config |
| `preset` | string | When enabled | Provider preset name (see below) |
| `secret` | string | When the preset needs one | Shared secret / token from the provider |
| `public_key` | string | When the preset needs one | Hex-encoded Ed25519 public key |

Enabling always replaces the stored config, so the secret must be sent on every
save — the API never echoes it back. See the
[authentication guide](/guides/authentication/) for how each preset verifies
requests.

## List authentication presets

```http
GET /api/auth/presets
```

Returns the selectable provider presets:

```json
[
  { "name": "github", "label": "GitHub", "needs_secret": true, "needs_public_key": false }
]
```

## Sanitized `auth_config`

Source responses never contain stored credentials. Instead of the raw config,
every source carries a sanitized view:

```json
{
  "auth_config": {
    "enabled": true,
    "scheme": "hmac",
    "preset": "github",
    "has_secret": true
  }
}
```
