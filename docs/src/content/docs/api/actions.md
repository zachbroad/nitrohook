---
title: Actions API
description: REST API reference for managing actions on a source.
---

Base path: `/api/sources/{slug}/actions`

## List actions

```http
GET /api/sources/{slug}/actions
```

Returns all actions attached to the source.

## Create action

```http
POST /api/sources/{slug}/actions
Content-Type: application/json

{
  "type": "webhook",
  "target_url": "https://example.com/hook",
  "is_active": true
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `type` | string | Yes | One of: `webhook`, `slack`, `smtp`, `twilio`, `javascript` |
| `target_url` | string | For webhook | Destination URL |
| `script_body` | string | For javascript | JavaScript source code |
| `signing_secret` | string | No | HMAC-SHA256 signing secret |
| `config` | object | For slack/smtp/twilio | Type-specific configuration (see [Action Types](/guides/action-types/)) |
| `transform_script` | string | No | Per-action transform script |
| `is_active` | boolean | No | Whether the action is active (default: true) |

## Get action

```http
GET /api/sources/{slug}/actions/{id}
```

## Update action

```http
PATCH /api/sources/{slug}/actions/{id}
Content-Type: application/json

{
  "is_active": false
}
```

## Delete action

```http
DELETE /api/sources/{slug}/actions/{id}
```
