---
title: Configuration
description: Environment variables and configuration options.
---

nitrohook is configured entirely through environment variables.

## Environment variables

| Variable | Default | Description |
|----------|---------|-------------|
| `DATABASE_URL` | `postgres://nitrohook:nitrohook@localhost:5432/nitrohook?sslmode=disable` | PostgreSQL connection string |
| `REDIS_URL` | `redis://localhost:6379` | Redis connection string |
| `PORT` | `8080` | HTTP server port |
| `WORKER_CONCURRENCY` | `4` | Number of concurrent stream consumers per worker |
| `MAX_RETRIES` | `5` | Maximum delivery attempts before marking as failed |
| `RETRY_BASE_DELAY` | `5s` | Base delay for exponential backoff (Go duration format) |
| `DELIVERY_TIMEOUT` | `10s` | HTTP timeout for outbound webhook requests |
| `POLL_INTERVAL` | `30s` | How often the worker polls for missed pending deliveries and retries |

## .env file

The API server loads a `.env` file from the working directory if present:

```bash
DATABASE_URL=postgres://nitrohook:nitrohook@localhost:5432/nitrohook?sslmode=disable
REDIS_URL=redis://localhost:6379
PORT=8080
WORKER_CONCURRENCY=4
MAX_RETRIES=5
RETRY_BASE_DELAY=5s
DELIVERY_TIMEOUT=10s
POLL_INTERVAL=30s
```

## Docker Compose

The default `docker-compose.yml` runs the full stack:

- **postgres** — data storage
- **redis** — delivery stream queue
- **migrate** — runs database migrations on startup
- **api** — HTTP server and web UI
- **worker** (2 replicas) — fan-out stream consumers

```bash
# Start everything
docker compose up -d

# Start only Postgres and Redis (for local Go development)
make docker-up-supporting-svc
```

## Running migrations

Migrations run automatically via the `migrate` service in Docker Compose. To run them manually:

```bash
# Via the API binary
go run ./cmd/api --migrate

# Via the migrate CLI
make migrate-up
```
