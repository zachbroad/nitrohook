---
name: verify
description: Build, launch, and drive NitroHook (Go API + React UI) locally to verify a change end-to-end.
---

# Verifying NitroHook changes locally

Host Postgres.app (5432) and Redis (6379) are usually running, but ports 8080
(api) and 5173 (vite) often have stale processes — always run an isolated
stack on alternate ports instead of reusing them.

## Isolated stack recipe

```bash
# 1. Isolated DB (create as superuser; the nitrohook role can't CREATE DATABASE)
psql -h localhost -d postgres -c "CREATE DATABASE nitrohook_verify OWNER nitrohook"
DATABASE_URL='postgres://nitrohook:nitrohook@localhost:5432/nitrohook_verify?sslmode=disable' \
  go run ./cmd/api --migrate

# 2. API + in-process worker on :8081 (use an isolated Redis DB index too)
#    CORS_ALLOWED_ORIGINS must include the vite origin or every browser
#    request fails preflight (default allows only http://localhost:5173).
DATABASE_URL='postgres://nitrohook:nitrohook@localhost:5432/nitrohook_verify?sslmode=disable' \
  REDIS_URL='redis://localhost:6379/5' PORT=8081 \
  CORS_ALLOWED_ORIGINS=http://localhost:5174 go run ./cmd/api --worker

# 3. React UI on :5174 pointed at the API
cd web/ui && VITE_API_BASE_URL=http://localhost:8081 npx vite --port 5174 --strictPort
```

Health check: `curl localhost:8081/readyz`. Seed data via the REST API
(`POST /api/sources` etc.), then drive `http://localhost:5174` with Playwright.

## Gotchas

- `go run` caches nothing: restart the API process after Go changes; vite
  hot-reloads UI changes but the SPA may bounce to `/` — re-navigate.
- Base UI (not Radix) components: the visible Switch is `role=switch`; the
  `id` lands on a hidden checkbox that Playwright can't click. Selects are
  `role=combobox` + `role=option`.
- Signed webhook for GitHub preset:
  `SIG=$(printf '%s' "$BODY" | openssl dgst -sha256 -hmac "$SECRET" -hex | awk '{print $NF}')`
  then header `X-Hub-Signature-256: sha256=$SIG`.
- Cleanup: stop both processes, `DROP DATABASE nitrohook_verify`.
