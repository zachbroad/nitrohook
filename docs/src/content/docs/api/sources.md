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
