---
title: Architecture
description: How nitrohook's components fit together.
---

## Overview

nitrohook has two main processes that can be deployed together or separately:

```
                    ┌─────────────────────────┐
                    │       API Server         │
  POST /webhooks/*  │                         │
  ────────────────► │  1. Store in Postgres    │
                    │  2. XADD to Redis Stream │
                    │  3. Return 202 Accepted  │
                    └────────────┬────────────┘
                                 │
                          Redis Stream
                        "deliveries"
                                 │
              ┌──────────────────┼──────────────────┐
              ▼                  ▼                   ▼
     ┌────────────────┐ ┌────────────────┐ ┌────────────────┐
     │   Worker 0     │ │   Worker 1     │ │   Worker N     │
     │  XREADGROUP    │ │  XREADGROUP    │ │  XREADGROUP    │
     │                │ │                │ │                │
     │  ► Transform   │ │  ► Transform   │ │  ► Transform   │
     │  ► Dispatch    │ │  ► Dispatch    │ │  ► Dispatch    │
     │  ► Record      │ │  ► Record      │ │  ► Record      │
     └────────────────┘ └────────────────┘ └────────────────┘
              │                  │                   │
              ▼                  ▼                   ▼
     ┌──────────────────────────────────────────────────────┐
     │                    PostgreSQL                         │
     │  sources │ actions │ deliveries │ delivery_attempts   │
     └──────────────────────────────────────────────────────┘
```

## Components

### API Server (`cmd/api`)

- Serves the web UI and REST API
- Handles webhook ingest at `POST /webhooks/{slug}`
- Writes deliveries to Postgres and publishes to the Redis Stream
- Can optionally run a worker in-process with `--worker` flag (useful for local dev)

### Fan-out Worker (`cmd/worker`)

- Consumes from the Redis Stream using consumer groups
- Runs N concurrent consumers (configurable via `WORKER_CONCURRENCY`)
- Executes source and action transform scripts
- Dispatches to action handlers (webhook, Slack, SMTP, Twilio, JS)
- Records delivery attempts in Postgres
- Handles retries with exponential backoff + jitter
- Background poll loops catch missed deliveries and process retries

### PostgreSQL

All state lives in Postgres: sources, actions, deliveries, delivery attempts. Postgres is the source of truth — if Redis loses a message, the poll loop picks up the pending delivery from Postgres.

### Redis

Used exclusively as a Redis Stream for the delivery work queue. The stream uses consumer groups (`fanout-workers`) so multiple worker replicas can process deliveries in parallel without duplicating work.

## Reliability

- Deliveries are written to Postgres **before** being published to Redis
- If the Redis `XADD` fails, the delivery is still safe — the worker's `pollPending` loop picks it up
- Workers ACK messages only **after** processing completes
- Failed dispatches are retried with exponential backoff (base delay * 2^attempt) with jitter
- Delivery status rolls up from individual attempt results across all actions

## Scaling

- **Workers**: add more replicas in `docker-compose.yml` — consumer groups handle distribution
- **API servers**: stateless, can be load-balanced
- **Postgres**: single instance is fine for most workloads; the schema supports connection pooling
