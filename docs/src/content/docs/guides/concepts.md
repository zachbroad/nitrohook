---
title: Concepts
description: Core concepts in nitrohook — sources, actions, deliveries, and the fan-out pipeline.
---

## Sources

A **source** is an ingest endpoint. Each source has a unique slug that becomes part of its webhook URL:

```
POST /webhooks/{slug}
```

Sources have a **mode**:

- **`active`** — webhooks are queued for immediate fan-out to all attached actions.
- **`record`** — webhooks are stored but not dispatched. You can inspect payloads and forward them manually when ready.

Sources can also have a **transform script** — a JavaScript function that runs on every delivery before dispatch. See [Scripting](/guides/scripting/).

## Actions

An **action** defines where and how a delivery gets dispatched. Actions are attached to a source — when a webhook arrives, all active actions on that source are triggered.

Each action has a **type**:

| Type | Description |
|------|-------------|
| `webhook` | Forward the payload to an HTTP endpoint |
| `slack` | Post to a Slack channel via incoming webhook |
| `smtp` | Send an email with the payload |
| `twilio` | Send an SMS via Twilio |
| `javascript` | Run a JavaScript function with the payload |

Actions can be toggled active/inactive without deleting them.

Each action can also have its own **transform script** that modifies the payload specifically for that action, or skips the action entirely by returning `null`.

See [Action Types](/guides/action-types/) for configuration details.

## Deliveries

A **delivery** is a single inbound webhook event. When a POST hits `/webhooks/{slug}`, nitrohook:

1. Stores the raw payload and headers in Postgres
2. Publishes the delivery ID to a Redis Stream
3. Returns `202 Accepted` with the delivery ID

Deliveries track their lifecycle with a status:

| Status | Meaning |
|--------|---------|
| `pending` | Queued, waiting for the worker |
| `processing` | Worker is dispatching to actions |
| `completed` | All actions succeeded (or were skipped) |
| `failed` | At least one action exhausted its retries |
| `recorded` | Stored in record mode, not dispatched |

## Delivery Attempts

Each action dispatch creates a **delivery attempt**. If an action fails, it retries with exponential backoff up to a configurable maximum (default: 5 attempts).

Attempts store the response status code, response body, and error message for debugging.

## Fan-out Pipeline

The full flow:

```
Webhook POST → Store in Postgres → XADD to Redis Stream
                                         ↓
                              Worker XREADGROUP
                                         ↓
                              Run source transform script (optional)
                                         ↓
                              For each active action:
                                → Run action transform script (optional)
                                → Dispatch via action type handler
                                → Record attempt result
                                         ↓
                              Roll up delivery status
```

Workers run as a consumer group, so you can scale horizontally by adding more worker replicas.
