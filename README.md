# NitroHook

Self-hosted webhook gateway that receives, transforms, and fans out webhook deliveries to multiple destinations. Built with Go, Postgres, and Redis.

## Screenshots

<table>
  <tr>
    <td width="50%"><img src="docs/screenshots/sources.png" alt="Sources list"><br><sub>Sources — each has a public ingest URL, a mode (active/record), and a set of fan-out actions.</sub></td>
    <td width="50%"><img src="docs/screenshots/actions.png" alt="Actions per source"><br><sub>Actions — webhook, Slack, SMTP, Twilio, or sandboxed JavaScript; toggle active/inactive inline.</sub></td>
  </tr>
  <tr>
    <td width="50%"><img src="docs/screenshots/script-editor.png" alt="Transform script editor"><br><sub>Monaco-powered transform script editor — mutate the payload, drop events by returning null, or filter <code>event.actions</code> to route selectively. Test against recorded payloads inline.</sub></td>
    <td width="50%"><img src="docs/screenshots/deliveries.png" alt="Events list"><br><sub>Events — every incoming webhook with its status (recorded / completed / failed) and idempotency key.</sub></td>
  </tr>
  <tr>
    <td colspan="2"><img src="docs/screenshots/delivery-detail.png" alt="Event detail with attempts"><br><sub>Event detail — raw headers, payload, and a per-attempt breakdown with HTTP status and errors.</sub></td>
  </tr>
</table>

## Features

- **Multi-destination fan-out** -- route incoming webhooks to HTTP endpoints, Slack, email (SMTP), Twilio SMS, or sandboxed JavaScript handlers
- **Payload transformation** -- attach JavaScript transform scripts at the source or per-action level to reshape payloads, filter events, or conditionally route deliveries
- **Retry with exponential backoff** -- failed deliveries are retried automatically with jitter-based backoff, per-attempt tracking, and configurable retry limits
- **HMAC signing** -- outbound webhook deliveries are signed with HMAC-SHA256 so downstream consumers can verify authenticity
- **Record mode** -- capture incoming webhooks without dispatching, then replay them on demand
- **Web UI and REST API** -- manage sources, actions, deliveries, and scripts through a browser UI or a JSON API
- **Prometheus metrics** -- built-in `/metrics` endpoint for ingest latency, dispatch duration, pending delivery counts, and retry queue depth

## Architecture

NitroHook runs as two cooperating processes backed by Postgres and Redis:

1. **API server** -- receives incoming webhooks at `POST /webhooks/:sourceSlug`, persists the payload and headers to Postgres, and publishes a delivery ID to a Redis Stream. Also serves the web UI, REST API, and Prometheus metrics.
2. **Worker** -- a pool of consumers reads from the Redis Stream consumer group, resolves the delivery's source and active actions, runs any configured transform scripts (via the Goja JS engine), and dispatches to each action type through a pluggable dispatcher registry. Failed attempts are scheduled for retry; a background poller picks up retryable attempts and pending deliveries as a catch-up mechanism.

Delivery status is rolled up across all actions: a delivery is marked completed only when every action succeeds or exhausts its retries.

## Tech Stack

- **Language:** Go 1.24
- **HTTP framework:** Gin
- **Database:** PostgreSQL (pgx driver, golang-migrate for schema management)
- **Queue:** Redis Streams (consumer groups for competing workers)
- **JS engine:** Goja (ES5+ runtime for transform scripts)
- **Metrics:** Prometheus client
- **Containerization:** Docker, Docker Compose

## Getting Started

### Docker Compose (recommended)

```bash
make docker-up      # starts Postgres, Redis, runs migrations, API, and 2 worker replicas
```

The API will be available at `http://localhost:8080`. To tear down:

```bash
make docker-down
```

### Local development

Prerequisites: Go 1.24+ and Docker.

For the fastest inner loop, use the one-command dev target:

```bash
make dev-setup   # once: installs Air (hot-reload tool)
make dev         # starts Postgres + Redis, migrates, runs API + worker with hot reload
```

`make dev` starts Postgres and Redis in Docker (waiting until they're healthy),
applies migrations via the API binary, then runs the API with an in-process
worker under [Air](https://github.com/air-verse/air). Editing any `.go` or
`.html` file triggers an automatic rebuild in ~1s. The API is available at
`http://localhost:8080`.

Press `Ctrl-C` to stop the app; the Postgres and Redis containers keep running
(so your data and startup time are preserved). To stop them too:

```bash
make dev-down    # stops the Postgres + Redis containers
```

#### Manual setup

If you'd rather run each piece yourself against your own Postgres/Redis:

```bash
# Create the database and role
make create-db

# Run migrations
make migrate-up

# Start the API server (optionally with an in-process worker)
make run-api            # API only
go run ./cmd/api --worker   # API + worker in one process

# Or start the worker separately
make run-worker
```

### Configuration

All settings are controlled via environment variables (or a `.env` file):

| Variable | Default | Description |
|---|---|---|
| `DATABASE_URL` | `postgres://nitrohook:nitrohook@localhost:5432/nitrohook?sslmode=disable` | Postgres connection string |
| `REDIS_URL` | `redis://localhost:6379` | Redis connection string |
| `PORT` | `8080` | HTTP listen port |
| `WORKER_CONCURRENCY` | `4` | Number of concurrent stream consumers per worker |
| `MAX_RETRIES` | `5` | Maximum delivery attempts per action |
| `RETRY_BASE_DELAY` | `5s` | Base delay for exponential backoff |
| `DELIVERY_TIMEOUT` | `10s` | HTTP client timeout for outbound dispatches |
| `POLL_INTERVAL` | `30s` | Interval for the pending/retry catch-up pollers |

### Running tests

```bash
make test                # unit tests
make test-integration    # integration tests (requires Postgres + Redis)
```

## CI/CD

Docker images are built and pushed to GHCR via `.github/workflows/docker-publish.yml` on every push to `main`. Deployment config (Helm chart, Terraform, environment values) lives in a separate private infra repo.
