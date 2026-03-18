# NitroHook

Webhook fan-out service in Go. Receives incoming webhooks via HTTP, stores them in Postgres, publishes to a Redis Stream, and a worker fans out deliveries to registered actions with retry logic, idempotency, and HMAC signing.

## Quick Start

```bash
make docker-up    # Postgres, Redis, API, 2 workers
make docker-down  # tear down
```

## CI/CD

Docker images are built and pushed to GHCR via `.github/workflows/docker-publish.yml` on every push to `main`. Deployment config (Helm chart, Terraform, environment values) lives in a separate private infra repo.
