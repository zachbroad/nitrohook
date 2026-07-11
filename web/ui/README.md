# NitroHook Admin UI

A React single-page app for managing NitroHook sources, actions, transform scripts, and deliveries. It talks to the Go API server's REST endpoints under `/api/`.

## Prerequisites

- Node.js (18+)
- The NitroHook API server running (see the [root README](../../README.md) for setup)

## Setup

```bash
cd web/ui
npm install
cp .env.example .env
```

`.env` sets `VITE_API_BASE_URL`, the base URL of the Go API server:

```
VITE_API_BASE_URL=http://localhost:8080
```

## Development

```bash
npm run dev
```

The app is served at [http://localhost:5173](http://localhost:5173).

**The Go API must allow this origin.** Start the API with `CORS_ALLOWED_ORIGINS` including `http://localhost:5173`, e.g.:

```bash
CORS_ALLOWED_ORIGINS=http://localhost:5173 make run-api
```

Without this, browser requests from the dev server to the API will fail CORS checks.

From the repo root, you can also use the Makefile shortcuts:

```bash
make ui-dev     # cd web/ui && npm run dev
make ui-build   # cd web/ui && npm run build
```

## Testing

```bash
npm run test
```

## Build

```bash
npm run build
```

Produces a production build in `web/ui/dist/`.
