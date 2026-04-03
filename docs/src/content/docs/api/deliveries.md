---
title: Deliveries API
description: REST API reference for querying deliveries and attempts.
---

Base path: `/api/deliveries`

## List deliveries

```http
GET /api/deliveries
```

Returns all deliveries, most recent first.

## Get delivery

```http
GET /api/deliveries/{id}
```

Returns a single delivery with its payload, headers, status, and transformed data (if any).

### Response

```json
{
  "id": "a1b2c3d4-...",
  "source_id": "...",
  "idempotency_key": "...",
  "headers": {"Content-Type": "application/json"},
  "payload": {"event": "push", "data": {}},
  "status": "completed",
  "received_at": "2025-01-15T10:30:00Z",
  "transformed_payload": null,
  "transformed_headers": null
}
```

### Delivery statuses

| Status | Description |
|--------|-------------|
| `pending` | Queued, waiting for worker pickup |
| `processing` | Worker is dispatching to actions |
| `completed` | All actions succeeded or were skipped |
| `failed` | At least one action exhausted all retries |
| `recorded` | Stored in record mode, not dispatched |

## List attempts

```http
GET /api/deliveries/{id}/attempts
```

Returns all dispatch attempts for a delivery, across all actions.

### Response

```json
[
  {
    "id": "...",
    "delivery_id": "...",
    "action_id": "...",
    "attempt_number": 1,
    "status": "success",
    "response_status": 200,
    "response_body": "{\"ok\":true}",
    "error_message": null,
    "next_retry_at": null,
    "created_at": "2025-01-15T10:30:01Z"
  }
]
```
