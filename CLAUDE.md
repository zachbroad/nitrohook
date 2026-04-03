# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What is NitroHook?

A webhook fan-out service in Go. Ingests webhooks via HTTP, stores them in PostgreSQL, publishes to Redis Streams, and fans out deliveries to registered actions (webhook, Slack, email, JavaScript, Twilio) with retry logic, HMAC signing, and script transformations.

## Build & Development Commands

```bash
make build                    # Build bin/api and bin/worker
make run-api                  # go run ./cmd/api
make run-worker               # go run ./cmd/worker
make test                     # go test ./...
make test-integration         # go test -tags=integration ./...
make docker-up-supporting-svc # Start Postgres + Redis only
make docker-up                # Full stack (Postgres, Redis, API, 2 workers)
make docker-down              # Tear down containers
make migrate-up               # Run pending migrations
make migrate-down             # Rollback migrations
```

Run a single test: `go test -run TestName ./internal/package/...`

Hot reload via Air: `air` (watches .go and .html files, configured in `.air.toml`)

## Architecture

Two binaries sharing the same internal packages:

- **`cmd/api`** — HTTP server (Gin). Ingests webhooks at `POST /webhooks/:sourceSlug`, serves REST API under `/api/`, and a web UI. Supports `--migrate` and `--worker` flags (in-process worker).
- **`cmd/worker`** — Standalone fan-out worker. Reads from Redis Stream `deliveries` (consumer group `fanout-workers`), dispatches to actions with exponential backoff retry.

### Key internal packages

| Package | Role |
|---------|------|
| `model` | Domain types: Source, Action, Delivery, DeliveryAttempt, enums |
| `store` | PostgreSQL data access (SourceStore, ActionStore, DeliveryStore) via pgx |
| `handler` | REST API handlers for webhooks, sources, actions, deliveries |
| `worker` | FanoutWorker — Redis consumer group, concurrent dispatch, retry polling |
| `dispatch` | Pluggable dispatchers (Webhook, Slack, SMTP, JavaScript, Twilio) implementing `Dispatcher` interface |
| `script` | Sandboxed JS execution via Goja (source transforms, action transforms, action scripts). 64KB limit, 500ms timeout |
| `signing` | HMAC-SHA256 signing/verification for webhook deliveries |
| `config` | Env-var-based config loading with defaults |
| `database` | pgx pool connection + golang-migrate runner |
| `web` | Web UI handlers + HTML templates |

### Data flow

1. Webhook received → stored in `deliveries` table → published to Redis Stream
2. Worker reads stream → fetches associated actions → runs source script (if any) → dispatches to each action
3. Each dispatch creates a `delivery_attempts` record; failures retry with exponential backoff

### Source modes

- **active** — fan-out to actions immediately
- **record** — store only, no fan-out (worker polls for these separately)

## Environment

Required services: PostgreSQL 18, Redis 8. See `.env.example` for all env vars. Key ones: `DATABASE_URL`, `REDIS_URL`, `PORT`, `WORKER_CONCURRENCY`, `MAX_RETRIES`.

## Migrations

SQL files in `/migrations/` managed by golang-migrate. Naming: sequential numbered pairs (`000001_name.up.sql` / `000001_name.down.sql`).
