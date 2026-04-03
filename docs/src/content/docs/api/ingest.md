---
title: Webhook Ingest
description: The ingest endpoint that receives incoming webhooks.
---

## Ingest a webhook

```http
POST /webhooks/{slug}
Content-Type: application/json

{"event": "deployment", "status": "success"}
```

This is the endpoint you give to third-party services (GitHub, Stripe, etc.) as your webhook URL.

### Headers

| Header | Description |
|--------|-------------|
| `X-Idempotency-Key` | Optional. Deduplicate deliveries. If omitted, a UUID is generated. |
| `Content-Type` | Must be `application/json`. |

### Response

```http
HTTP/1.1 202 Accepted

{
  "delivery_id": "a1b2c3d4-...",
  "status": "pending"
}
```

In **record mode**, the status will be `"recorded"` instead of `"pending"`, and no fan-out occurs.

### What happens next

1. The payload and headers are stored in Postgres
2. The delivery ID is published to a Redis Stream (`XADD`)
3. A fan-out worker picks it up (`XREADGROUP`)
4. The source transform script runs (if configured)
5. Each active action is dispatched
6. Delivery attempts are recorded with response details

If the Redis publish fails, the delivery is still safe in Postgres. A background poll loop catches any pending deliveries that weren't picked up from the stream.
