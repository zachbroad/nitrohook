---
title: Quickstart
description: Get nitrohook running locally in under a minute.
---

## With Docker Compose

The fastest way to get started. This spins up Postgres, Redis, the API server, and two worker replicas.

```bash
git clone https://github.com/zachbroad/nitrohook.git
cd nitrohook
docker compose up -d
```

The API and web UI will be available at `http://localhost:8080`.

### Verify it's running

```bash
curl http://localhost:8080/healthz
# .

curl http://localhost:8080/readyz
# {"status":"ok"}
```

## Without Docker

### Prerequisites

- Go 1.24+
- PostgreSQL 16+
- Redis 7+

### Setup

```bash
# Create the database
make create-db

# Run migrations
go run ./cmd/api --migrate

# Start supporting services (if using docker for postgres/redis only)
make docker-up-supporting-svc

# Start the API server (with an in-process worker)
go run ./cmd/api --worker
```

## Send your first webhook

### 1. Create a source

Open `http://localhost:8080/sources` in your browser and create a new source, or use the API:

```bash
curl -X POST http://localhost:8080/api/sources \
  -H "Content-Type: application/json" \
  -d '{"name": "My App", "slug": "my-app"}'
```

### 2. Add an action

Add a webhook action that forwards to a test endpoint:

```bash
curl -X POST http://localhost:8080/api/sources/my-app/actions \
  -H "Content-Type: application/json" \
  -d '{
    "type": "webhook",
    "target_url": "https://httpbin.org/post",
    "is_active": true
  }'
```

### 3. Send a webhook

```bash
curl -X POST http://localhost:8080/webhooks/my-app \
  -H "Content-Type: application/json" \
  -d '{"event": "test", "data": {"message": "hello from nitrohook"}}'
```

You'll get back a delivery ID:

```json
{"delivery_id": "a1b2c3d4-...", "status": "pending"}
```

The worker picks it up, forwards it to httpbin, and records the result. Check the delivery in the web UI at `http://localhost:8080/deliveries`.
