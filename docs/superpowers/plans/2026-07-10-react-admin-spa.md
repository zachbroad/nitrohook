# NitroHook React Admin SPA Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a two-pane master–detail admin SPA (Vite + React Router v7 + shadcn/ui) that fully replaces the Go-template UI, consuming the existing `/api` JSON endpoints plus three new ones.

**Architecture:** The SPA in `web/ui/` is a standalone client of the Go `/api`. Phase A adds a small Go backend delta (CORS + 3 JSON endpoints reusing existing store/script/publish logic). Phases B–E build the React app: infra (client, types, query hooks, shell), then Sources, then Deliveries, then polish.

**Tech Stack:** Go 1.x + Gin (backend delta); Vite + React 18 + TypeScript + React Router v7 + Tailwind + shadcn/ui + TanStack Query + react-hook-form + zod + CodeMirror 6 + sonner; Vitest + React Testing Library + MSW (tests).

## Global Constraints

- SPA lives in `web/ui/`; served standalone (no `go:embed`). API base URL from `VITE_API_BASE_URL` (default `http://localhost:8080`).
- No authentication. CORS allowed origins from Go env `CORS_ALLOWED_ORIGINS` (comma-separated, default `http://localhost:5173`).
- Go module path: `github.com/zachbroad/nitrohook`. New JSON handlers go in `internal/handler`, wired in `cmd/api/main.go` under the existing `/api` group.
- The Go-template UI (`web/`) must remain functional and untouched.
- Action types are exactly: `webhook`, `javascript`, `slack`, `smtp`, `twilio`.
- Dispatcher `config` JSON field names (verbatim): Slack `{webhook_url, channel?, username?}`; SMTP `{host, port, username, password, from, to, subject}`; Twilio `{account_sid, auth_token, from, to, body_template?}`.
- Delivery statuses: `pending, processing, completed, failed, recorded`. Attempt statuses: `pending, success, failed`. Source modes: `active, record`.
- Redis stream publish for a forward: `XAdd` to stream `deliveries`, values `{delivery_id: <uuid>, force: "1"}`, `MaxLen 10000 Approx`.
- TDD: write failing test → confirm fail → implement → confirm pass → commit. Commit after each task.

---

## Phase A — Backend delta (Go)

### Task 1: CORS middleware + config

**Files:**
- Create: `internal/middleware/cors.go`
- Create: `internal/middleware/cors_test.go`
- Modify: `internal/config/config.go` (add `CORSAllowedOrigins []string`)
- Modify: `cmd/api/main.go` (register middleware after `RequestLogger`)

**Interfaces:**
- Produces: `middleware.CORS(allowedOrigins []string) gin.HandlerFunc`
- Consumes: existing `config.Load()` returning `*config.Config`.

- [ ] **Step 1: Write the failing test** — `internal/middleware/cors_test.go`

```go
package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func setupCORS(t *testing.T) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(CORS([]string{"http://localhost:5173"}))
	r.GET("/x", func(c *gin.Context) { c.String(200, "ok") })
	return r
}

func TestCORS_AllowsListedOrigin(t *testing.T) {
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/x", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	setupCORS(t).ServeHTTP(w, req)
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:5173" {
		t.Fatalf("allow-origin = %q, want http://localhost:5173", got)
	}
}

func TestCORS_PreflightReturns204(t *testing.T) {
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("OPTIONS", "/x", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	req.Header.Set("Access-Control-Request-Method", "PATCH")
	setupCORS(t).ServeHTTP(w, req)
	if w.Code != http.StatusNoContent {
		t.Fatalf("preflight status = %d, want 204", w.Code)
	}
}

func TestCORS_UnlistedOriginNoHeader(t *testing.T) {
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/x", nil)
	req.Header.Set("Origin", "http://evil.example")
	setupCORS(t).ServeHTTP(w, req)
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("allow-origin = %q, want empty", got)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/middleware/ -run TestCORS -v`
Expected: FAIL — `undefined: CORS`.

- [ ] **Step 3: Write minimal implementation** — `internal/middleware/cors.go`

```go
package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// CORS allows cross-origin requests from the listed origins. Reflects the
// request Origin only when it is in the allow-list, and answers preflight
// OPTIONS requests with 204.
func CORS(allowedOrigins []string) gin.HandlerFunc {
	allowed := make(map[string]bool, len(allowedOrigins))
	for _, o := range allowedOrigins {
		allowed[o] = true
	}
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if origin != "" && allowed[origin] {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Vary", "Origin")
			c.Header("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
			c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Request-ID")
			c.Header("Access-Control-Max-Age", "600")
		}
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/middleware/ -run TestCORS -v`
Expected: PASS (3 tests).

- [ ] **Step 5: Wire config + main.go**

In `internal/config/config.go` add to the `Config` struct field `CORSAllowedOrigins []string` and in `Load()` populate it by splitting env `CORS_ALLOWED_ORIGINS` on `,` (trim spaces); default `[]string{"http://localhost:5173"}` when unset. Match the file's existing getenv/default helper style.

In `cmd/api/main.go`, immediately after `r.Use(middleware.RequestLogger())` add:

```go
	r.Use(middleware.CORS(cfg.CORSAllowedOrigins))
```

- [ ] **Step 6: Verify build + tests**

Run: `go build ./... && go test ./internal/middleware/ ./internal/config/`
Expected: build succeeds, tests PASS.

- [ ] **Step 7: Commit**

```bash
git add internal/middleware/cors.go internal/middleware/cors_test.go internal/config/config.go cmd/api/main.go
git commit -m "feat(api): add configurable CORS middleware"
```

---

### Task 2: JSON script-test endpoint

Reuses `script.Run` (returns `*script.TransformResult` which already has JSON tags: `{payload, headers, actions, dropped}`) and `store.Actions.ListActiveBySource`.

**Files:**
- Modify: `internal/handler/source.go` (add `TestScript` method)
- Create: `internal/handler/source_script_test.go`
- Modify: `cmd/api/main.go` (register route)

**Interfaces:**
- Produces: route `POST /api/sources/:sourceSlug/script/test`. Request JSON `{ "script_body": string, "delivery_id": string }`. Success `200` JSON `{ "result": {payload, headers, actions, dropped}, "error": null }`. Script error → `200` JSON `{ "result": null, "error": "<message>" }`. Bad input → `400` plain text.
- Consumes: `h.store.Sources.GetBySlug`, `h.store.Deliveries.GetByID`, `h.store.Actions.ListActiveBySource`, `script.Run`, `script.TransformInput`, `script.ActionRef`.

- [ ] **Step 1: Write the failing test** — `internal/handler/source_script_test.go`

```go
package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

// TestScript_BadBody exercises validation without a DB: an empty JSON body
// must be rejected with 400 before any store access.
func TestScript_BadBody(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := &SourceHandler{} // store unused on the validation path
	r := gin.New()
	r.POST("/api/sources/:sourceSlug/script/test", h.TestScript)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/sources/demo/script/test", strings.NewReader(`not json`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}

// scriptTestResponse documents the success envelope shape for consumers.
type scriptTestResponse struct {
	Result json.RawMessage `json:"result"`
	Error  *string         `json:"error"`
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/handler/ -run TestScript_BadBody -v`
Expected: FAIL — `h.TestScript undefined`.

- [ ] **Step 3: Write minimal implementation** — add to `internal/handler/source.go`

Add imports as needed (`encoding/json`, `net/http`, `github.com/google/uuid`, `github.com/zachbroad/nitrohook/internal/script`). Method:

```go
type testScriptRequest struct {
	ScriptBody string `json:"script_body"`
	DeliveryID string `json:"delivery_id"`
}

// TestScript runs a candidate source-transform script against a recorded
// delivery and returns the transform result as JSON. It never returns 500 for
// a script that merely throws — that surfaces as {"result":null,"error":...}.
func (h *SourceHandler) TestScript(c *gin.Context) {
	slug := c.Param("sourceSlug")
	var req testScriptRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.String(http.StatusBadRequest, "invalid request body")
		return
	}
	if strings.TrimSpace(req.DeliveryID) == "" {
		c.String(http.StatusBadRequest, "delivery_id is required")
		return
	}
	if strings.TrimSpace(req.ScriptBody) == "" {
		c.String(http.StatusBadRequest, "script_body is required")
		return
	}

	source, err := h.store.Sources.GetBySlug(c.Request.Context(), slug)
	if err != nil {
		c.String(http.StatusNotFound, "source not found")
		return
	}
	did, err := uuid.Parse(req.DeliveryID)
	if err != nil {
		c.String(http.StatusBadRequest, "invalid delivery_id")
		return
	}
	delivery, err := h.store.Deliveries.GetByID(c.Request.Context(), did)
	if err != nil || delivery.SourceID != source.ID {
		c.String(http.StatusNotFound, "delivery not found for this source")
		return
	}

	var payload map[string]any
	if delivery.Payload != nil {
		if err := json.Unmarshal(delivery.Payload, &payload); err != nil {
			payload = map[string]any{"_raw": string(delivery.Payload)}
		}
	}
	var headers map[string]string
	if delivery.Headers != nil {
		_ = json.Unmarshal(delivery.Headers, &headers)
	}

	actions, _ := h.store.Actions.ListActiveBySource(c.Request.Context(), source.ID)
	actionRefs := make([]script.ActionRef, len(actions))
	for i, a := range actions {
		targetURL := ""
		if a.TargetURL != nil {
			targetURL = *a.TargetURL
		}
		actionRefs[i] = script.ActionRef{ID: a.ID, TargetURL: targetURL}
	}

	result, err := script.Run(req.ScriptBody, script.TransformInput{
		Payload: payload, Headers: headers, Actions: actionRefs,
	})
	if err != nil {
		msg := err.Error()
		c.JSON(http.StatusOK, gin.H{"result": nil, "error": msg})
		return
	}
	c.JSON(http.StatusOK, gin.H{"result": result, "error": nil})
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/handler/ -run TestScript_BadBody -v`
Expected: PASS.

- [ ] **Step 5: Register route** in `cmd/api/main.go` inside the `srcGroup` block (next to `srcGroup.GET/PATCH/DELETE`):

```go
				srcGroup.POST("/script/test", sourceH.TestScript)
```

- [ ] **Step 6: Verify build**

Run: `go build ./... && go test ./internal/handler/ -run TestScript`
Expected: build succeeds, PASS.

- [ ] **Step 7: Commit**

```bash
git add internal/handler/source.go internal/handler/source_script_test.go cmd/api/main.go
git commit -m "feat(api): add JSON source script-test endpoint"
```

---

### Task 3: JSON forward-delivery + forward-all endpoints

Mirrors `web.ForwardDelivery` / `web.ForwardAllRecorded` but returns JSON. The delivery handler needs Redis to publish; extend `DeliveryHandler` with an `rdb *redis.Client` field.

**Files:**
- Modify: `internal/handler/delivery.go` (add `rdb`, `NewDeliveryHandlerWithRedis` or extend constructor, `Forward`, publish helper)
- Modify: `internal/handler/source.go` (add `ForwardAll` — or place on DeliveryHandler; keep on DeliveryHandler for the publish helper). Put both on `DeliveryHandler`.
- Create: `internal/handler/delivery_forward_test.go`
- Modify: `cmd/api/main.go` (pass `rdb` to delivery handler; register routes)

**Interfaces:**
- Produces:
  - `POST /api/deliveries/:id/forward` → `200 {"status":"forwarded"}`; non-recorded → `200 {"status":"skipped"}`; not found → `404`.
  - `POST /api/sources/:sourceSlug/deliveries/forward-all` → `200 {"forwarded": <int>}`.
  - Constructor `handler.NewDeliveryHandler(s *store.Store, rdb *redis.Client) *DeliveryHandler` (add the `rdb` param).
- Consumes: `store.Deliveries.GetByID/List/UpdateStatus`, `store.Sources.GetBySlug`, `redis XAdd`.

- [ ] **Step 1: Write the failing test** — `internal/handler/delivery_forward_test.go`

```go
package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

// TestForward_BadID checks the pre-store validation path: a non-UUID id is
// rejected with 400 without needing Redis or the DB.
func TestForward_BadID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := &DeliveryHandler{}
	r := gin.New()
	r.POST("/api/deliveries/:id/forward", h.Forward)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/deliveries/not-a-uuid/forward", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/handler/ -run TestForward_BadID -v`
Expected: FAIL — `h.Forward undefined`.

- [ ] **Step 3: Write minimal implementation**

In `internal/handler/delivery.go`: add field `rdb *redis.Client` to `DeliveryHandler`, update `NewDeliveryHandler(s *store.Store, rdb *redis.Client)` to set it, and add (import `context`, `github.com/google/uuid`, `github.com/redis/go-redis/v9`, `github.com/zachbroad/nitrohook/internal/model`, `log/slog`):

```go
func (h *DeliveryHandler) publishForward(ctx context.Context, id uuid.UUID) error {
	return h.rdb.XAdd(ctx, &redis.XAddArgs{
		Stream: "deliveries", MaxLen: 10000, Approx: true,
		Values: map[string]any{"delivery_id": id.String(), "force": "1"},
	}).Err()
}

// Forward re-queues a single recorded delivery for fan-out.
func (h *DeliveryHandler) Forward(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.String(http.StatusBadRequest, "invalid delivery id")
		return
	}
	d, err := h.store.Deliveries.GetByID(c.Request.Context(), id)
	if err != nil {
		c.String(http.StatusNotFound, "delivery not found")
		return
	}
	if d.Status != model.DeliveryRecorded {
		c.JSON(http.StatusOK, gin.H{"status": "skipped"})
		return
	}
	if err := h.store.Deliveries.UpdateStatus(c.Request.Context(), id, model.DeliveryPending); err != nil {
		c.String(http.StatusInternalServerError, "failed to forward delivery")
		return
	}
	if err := h.publishForward(c.Request.Context(), id); err != nil {
		slog.Error("forward publish failed", "error", err, "delivery_id", id)
		_ = h.store.Deliveries.UpdateStatus(c.Request.Context(), id, model.DeliveryRecorded)
		c.String(http.StatusInternalServerError, "failed to forward delivery")
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "forwarded"})
}

// ForwardAll re-queues every recorded delivery for a source.
func (h *DeliveryHandler) ForwardAll(c *gin.Context) {
	slug := c.Param("sourceSlug")
	if _, err := h.store.Sources.GetBySlug(c.Request.Context(), slug); err != nil {
		c.String(http.StatusNotFound, "source not found")
		return
	}
	deliveries, err := h.store.Deliveries.List(c.Request.Context(), &slug, 200)
	if err != nil {
		c.String(http.StatusInternalServerError, "failed to list deliveries")
		return
	}
	forwarded := 0
	for _, d := range deliveries {
		if d.Status != model.DeliveryRecorded {
			continue
		}
		if err := h.store.Deliveries.UpdateStatus(c.Request.Context(), d.ID, model.DeliveryPending); err != nil {
			continue
		}
		if err := h.publishForward(c.Request.Context(), d.ID); err != nil {
			_ = h.store.Deliveries.UpdateStatus(c.Request.Context(), d.ID, model.DeliveryRecorded)
			continue
		}
		forwarded++
	}
	c.JSON(http.StatusOK, gin.H{"forwarded": forwarded})
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/handler/ -run TestForward_BadID -v`
Expected: PASS.

- [ ] **Step 5: Wire main.go** — update the delivery handler construction to `deliveryH := handler.NewDeliveryHandler(s, rdb)` and add routes:

```go
		// inside srcGroup:
		srcGroup.POST("/deliveries/forward-all", deliveryH.ForwardAll)
		// inside deliveries group:
		deliveries.POST("/:id/forward", deliveryH.Forward)
```

Also update `internal/handler/handler_integration_test.go` where `NewDeliveryHandler` is called to pass a redis client (use the existing test redis if present, else `nil` where the tested path doesn't publish). Verify that file still compiles.

- [ ] **Step 6: Verify build + full handler tests**

Run: `go build ./... && go test ./internal/handler/`
Expected: build succeeds; tests PASS (skip integration if `-tags=integration` not set).

- [ ] **Step 7: Commit**

```bash
git add internal/handler/delivery.go internal/handler/delivery_forward_test.go internal/handler/handler_integration_test.go cmd/api/main.go
git commit -m "feat(api): add JSON forward-delivery and forward-all endpoints"
```

---

## Phase B — Frontend infrastructure

### Task 4: Scaffold Vite + Tailwind + shadcn + test harness

**Files:**
- Create: `web/ui/` project (Vite React-TS template), `web/ui/.env.example`, `web/ui/tailwind.config.ts`, `web/ui/src/index.css`, `web/ui/components.json`, `web/ui/vitest.config.ts`, `web/ui/src/test/setup.ts`, `web/ui/.gitignore`
- Modify: `Makefile` (add `ui-dev`, `ui-build` targets)

**Interfaces:**
- Produces: a runnable dev server (`npm run dev` in `web/ui`), Tailwind + shadcn configured, `@/` path alias → `src/`, Vitest+RTL+MSW ready. `import.meta.env.VITE_API_BASE_URL` available.

- [ ] **Step 1: Scaffold the app**

```bash
cd web && npm create vite@latest ui -- --template react-ts && cd ui && npm install
```

- [ ] **Step 2: Install dependencies**

```bash
cd web/ui
npm install react-router-dom @tanstack/react-query react-hook-form @hookform/resolvers zod \
  sonner lucide-react clsx tailwind-merge class-variance-authority \
  @uiw/react-codemirror @codemirror/lang-javascript @codemirror/lang-json
npm install -D tailwindcss @tailwindcss/vite vitest @testing-library/react \
  @testing-library/jest-dom @testing-library/user-event jsdom msw
```

- [ ] **Step 3: Configure Vite + alias + Tailwind** — `web/ui/vite.config.ts`

```ts
import path from "path"
import { defineConfig } from "vite"
import react from "@vitejs/plugin-react"
import tailwindcss from "@tailwindcss/vite"

export default defineConfig({
  plugins: [react(), tailwindcss()],
  resolve: { alias: { "@": path.resolve(__dirname, "./src") } },
  server: { port: 5173 },
})
```

Replace `web/ui/src/index.css` with `@import "tailwindcss";` plus shadcn CSS variables (theme tokens). Add `web/ui/src/vite-env.d.ts` augmentation:

```ts
interface ImportMetaEnv { readonly VITE_API_BASE_URL: string }
interface ImportMeta { readonly env: ImportMetaEnv }
```

Set `tsconfig.json` / `tsconfig.app.json` `compilerOptions.baseUrl: "."` and `paths: { "@/*": ["./src/*"] }`.

- [ ] **Step 4: Init shadcn + add base components**

```bash
cd web/ui
npx shadcn@latest init -d
npx shadcn@latest add button dialog input label select textarea table tabs badge \
  card dropdown-menu sonner skeleton switch tooltip separator scroll-area
```

- [ ] **Step 5: Configure Vitest** — `web/ui/vitest.config.ts`

```ts
import { defineConfig } from "vitest/config"
import react from "@vitejs/plugin-react"
import path from "path"

export default defineConfig({
  plugins: [react()],
  resolve: { alias: { "@": path.resolve(__dirname, "./src") } },
  test: { environment: "jsdom", globals: true, setupFiles: ["./src/test/setup.ts"] },
})
```

`web/ui/src/test/setup.ts`:

```ts
import "@testing-library/jest-dom/vitest"
```

Add `web/ui/.env.example` with `VITE_API_BASE_URL=http://localhost:8080`. Add `"test": "vitest run"` and `"test:watch": "vitest"` to `package.json` scripts.

- [ ] **Step 6: Smoke test** — create `web/ui/src/App.test.tsx`:

```tsx
import { render, screen } from "@testing-library/react"
import { expect, test } from "vitest"

test("harness renders", () => {
  render(<h1>NitroHook</h1>)
  expect(screen.getByText("NitroHook")).toBeInTheDocument()
})
```

Run: `cd web/ui && npm run test`
Expected: 1 passing test.

- [ ] **Step 7: Makefile targets** — add:

```make
ui-dev: ## Run the React admin SPA dev server
	cd web/ui && npm run dev

ui-build: ## Build the React admin SPA
	cd web/ui && npm run build
```

- [ ] **Step 8: Commit**

```bash
git add web/ui Makefile
git commit -m "chore(ui): scaffold Vite + React + Tailwind + shadcn + Vitest"
```

---

### Task 5: Types + typed API client

**Files:**
- Create: `web/ui/src/lib/types.ts`, `web/ui/src/lib/api.ts`, `web/ui/src/lib/api.test.ts`

**Interfaces:**
- Produces:
  - `types.ts`: `Source`, `ActionType`, `Action`, `DeliveryStatus`, `Delivery`, `AttemptStatus`, `DeliveryAttempt`, `ScriptTestResult`, `TransformResult`.
  - `api.ts`: `class ApiError extends Error { status: number }`, `apiFetch<T>(path, opts?): Promise<T>`, and endpoint fns: `listSources()`, `getSource(slug)`, `createSource(body)`, `updateSource(slug, body)`, `deleteSource(slug)`, `listActions(slug)`, `createAction(slug, body)`, `updateAction(slug, id, body)`, `deleteAction(slug, id)`, `testScript(slug, body)`, `listDeliveries(params)`, `getDelivery(id)`, `listAttempts(id)`, `forwardDelivery(id)`, `forwardAll(slug)`.

- [ ] **Step 1: Write types** — `web/ui/src/lib/types.ts` (mirror Go structs verbatim, snake_case keys)

```ts
export interface Source {
  id: string; name: string; slug: string; mode: "active" | "record";
  script_body?: string | null; created_at: string; updated_at: string;
}
export type ActionType = "webhook" | "javascript" | "slack" | "smtp" | "twilio";
export interface Action {
  id: string; source_id: string; type: ActionType;
  target_url?: string | null; script_body?: string | null; signing_secret?: string | null;
  config?: unknown; transform_script?: string | null; is_active: boolean;
  created_at: string; updated_at: string;
}
export type DeliveryStatus = "pending" | "processing" | "completed" | "failed" | "recorded";
export interface Delivery {
  id: string; source_id: string; idempotency_key: string;
  headers: unknown; payload: unknown; status: DeliveryStatus; received_at: string;
  transformed_payload?: unknown; transformed_headers?: unknown;
}
export type AttemptStatus = "pending" | "success" | "failed";
export interface DeliveryAttempt {
  id: string; delivery_id: string; action_id: string; attempt_number: number;
  status: AttemptStatus; response_status?: number | null; response_body?: string | null;
  error_message?: string | null; next_retry_at?: string | null; created_at: string;
}
export interface TransformResult {
  payload: Record<string, unknown>; headers: Record<string, string>;
  actions: { id: string; target_url: string }[]; dropped: boolean;
}
export interface ScriptTestResult { result: TransformResult | null; error: string | null }
```

- [ ] **Step 2: Write the failing test** — `web/ui/src/lib/api.test.ts`

```ts
import { describe, expect, it, vi, beforeEach } from "vitest"
import { apiFetch, ApiError } from "./api"

beforeEach(() => { vi.restoreAllMocks() })

describe("apiFetch", () => {
  it("returns parsed JSON on 200", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(
      new Response(JSON.stringify({ ok: true }), { status: 200, headers: { "Content-Type": "application/json" } })
    ))
    await expect(apiFetch<{ ok: boolean }>("/api/x")).resolves.toEqual({ ok: true })
  })

  it("throws ApiError with server text on non-2xx", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(new Response("boom", { status: 400 })))
    await expect(apiFetch("/api/x")).rejects.toMatchObject({ status: 400, message: "boom" })
    await expect(apiFetch("/api/x")).rejects.toBeInstanceOf(ApiError)
  })
})
```

- [ ] **Step 3: Run test to verify it fails**

Run: `cd web/ui && npx vitest run src/lib/api.test.ts`
Expected: FAIL — cannot import `./api`.

- [ ] **Step 4: Implement** — `web/ui/src/lib/api.ts`

```ts
import type {
  Source, Action, Delivery, DeliveryAttempt, ScriptTestResult, ActionType,
} from "./types"

const BASE = import.meta.env.VITE_API_BASE_URL ?? "http://localhost:8080"

export class ApiError extends Error {
  status: number
  constructor(status: number, message: string) { super(message); this.status = status; this.name = "ApiError" }
}

export async function apiFetch<T>(path: string, opts: RequestInit = {}): Promise<T> {
  const res = await fetch(BASE + path, {
    ...opts,
    headers: { "Content-Type": "application/json", ...(opts.headers ?? {}) },
  })
  if (!res.ok) throw new ApiError(res.status, (await res.text()) || res.statusText)
  if (res.status === 204) return undefined as T
  const ct = res.headers.get("Content-Type") ?? ""
  return (ct.includes("application/json") ? await res.json() : (await res.text())) as T
}

const j = (body: unknown): RequestInit => ({ body: JSON.stringify(body) })

export const listSources = () => apiFetch<Source[]>("/api/sources")
export const getSource = (slug: string) => apiFetch<Source>(`/api/sources/${slug}`)
export const createSource = (b: { name: string; mode?: string; script_body?: string }) =>
  apiFetch<Source>("/api/sources", { method: "POST", ...j(b) })
export const updateSource = (slug: string, b: Partial<Pick<Source, "mode" | "script_body">>) =>
  apiFetch<Source>(`/api/sources/${slug}`, { method: "PATCH", ...j(b) })
export const deleteSource = (slug: string) =>
  apiFetch<void>(`/api/sources/${slug}`, { method: "DELETE" })

export const listActions = (slug: string) => apiFetch<Action[]>(`/api/sources/${slug}/actions`)
export const createAction = (slug: string, b: Record<string, unknown> & { type: ActionType }) =>
  apiFetch<Action>(`/api/sources/${slug}/actions`, { method: "POST", ...j(b) })
export const updateAction = (slug: string, id: string, b: Record<string, unknown>) =>
  apiFetch<Action>(`/api/sources/${slug}/actions/${id}`, { method: "PATCH", ...j(b) })
export const deleteAction = (slug: string, id: string) =>
  apiFetch<void>(`/api/sources/${slug}/actions/${id}`, { method: "DELETE" })

export const testScript = (slug: string, b: { script_body: string; delivery_id: string }) =>
  apiFetch<ScriptTestResult>(`/api/sources/${slug}/script/test`, { method: "POST", ...j(b) })

export const listDeliveries = (p: { source?: string; limit?: number } = {}) => {
  const q = new URLSearchParams()
  if (p.source) q.set("source", p.source)
  if (p.limit) q.set("limit", String(p.limit))
  const qs = q.toString()
  return apiFetch<Delivery[]>(`/api/deliveries${qs ? `?${qs}` : ""}`)
}
export const getDelivery = (id: string) => apiFetch<Delivery>(`/api/deliveries/${id}`)
export const listAttempts = (id: string) => apiFetch<DeliveryAttempt[]>(`/api/deliveries/${id}/attempts`)
export const forwardDelivery = (id: string) =>
  apiFetch<{ status: string }>(`/api/deliveries/${id}/forward`, { method: "POST" })
export const forwardAll = (slug: string) =>
  apiFetch<{ forwarded: number }>(`/api/sources/${slug}/deliveries/forward-all`, { method: "POST" })
```

- [ ] **Step 5: Run test to verify it passes**

Run: `cd web/ui && npx vitest run src/lib/api.test.ts`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add web/ui/src/lib/types.ts web/ui/src/lib/api.ts web/ui/src/lib/api.test.ts
git commit -m "feat(ui): typed API client mirroring Go models"
```

---

### Task 6: Query hooks + QueryClient

**Files:**
- Create: `web/ui/src/lib/queries.ts`
- Create: `web/ui/src/lib/utils.ts` (if `shadcn init` did not already create `cn`)

**Interfaces:**
- Produces: query-key factory `qk` and hooks `useSources`, `useSource(slug)`, `useActions(slug)`, `useDeliveries(params)`, `useDelivery(id)`, `useAttempts(id)`; mutation hooks `useCreateSource`, `useUpdateSource`, `useDeleteSource`, `useCreateAction`, `useUpdateAction`, `useDeleteAction`, `useForwardDelivery`, `useForwardAll`. Each mutation invalidates the affected keys and toasts on error.
- Consumes: everything from `api.ts`.

- [ ] **Step 1: Implement** — `web/ui/src/lib/queries.ts`

```ts
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { toast } from "sonner"
import * as api from "./api"
import { ApiError } from "./api"

export const qk = {
  sources: ["sources"] as const,
  source: (s: string) => ["source", s] as const,
  actions: (s: string) => ["source", s, "actions"] as const,
  deliveries: (p: { source?: string } = {}) => ["deliveries", p] as const,
  delivery: (id: string) => ["delivery", id] as const,
  attempts: (id: string) => ["delivery", id, "attempts"] as const,
}

const onError = (e: unknown) =>
  toast.error(e instanceof ApiError ? e.message : "Request failed")

export const useSources = () => useQuery({ queryKey: qk.sources, queryFn: api.listSources })
export const useSource = (slug: string) =>
  useQuery({ queryKey: qk.source(slug), queryFn: () => api.getSource(slug), enabled: !!slug })
export const useActions = (slug: string) =>
  useQuery({ queryKey: qk.actions(slug), queryFn: () => api.listActions(slug), enabled: !!slug })
export const useDeliveries = (p: { source?: string } = {}) =>
  useQuery({ queryKey: qk.deliveries(p), queryFn: () => api.listDeliveries(p) })
export const useDelivery = (id: string) =>
  useQuery({ queryKey: qk.delivery(id), queryFn: () => api.getDelivery(id), enabled: !!id })
export const useAttempts = (id: string) =>
  useQuery({ queryKey: qk.attempts(id), queryFn: () => api.listAttempts(id), enabled: !!id })

export function useCreateSource() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: api.createSource, onError,
    onSuccess: () => qc.invalidateQueries({ queryKey: qk.sources }),
  })
}
export function useUpdateSource(slug: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (b: Parameters<typeof api.updateSource>[1]) => api.updateSource(slug, b), onError,
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: qk.source(slug) })
      qc.invalidateQueries({ queryKey: qk.sources })
    },
  })
}
export function useDeleteSource() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: api.deleteSource, onError,
    onSuccess: () => qc.invalidateQueries({ queryKey: qk.sources }),
  })
}
export function useCreateAction(slug: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (b: Parameters<typeof api.createAction>[1]) => api.createAction(slug, b), onError,
    onSuccess: () => qc.invalidateQueries({ queryKey: qk.actions(slug) }),
  })
}
export function useUpdateAction(slug: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (v: { id: string; body: Record<string, unknown> }) => api.updateAction(slug, v.id, v.body), onError,
    onSuccess: () => qc.invalidateQueries({ queryKey: qk.actions(slug) }),
  })
}
export function useDeleteAction(slug: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => api.deleteAction(slug, id), onError,
    onSuccess: () => qc.invalidateQueries({ queryKey: qk.actions(slug) }),
  })
}
export function useForwardDelivery(slug?: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: api.forwardDelivery,
    onError,
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["deliveries"] })
      if (slug) qc.invalidateQueries({ queryKey: qk.deliveries({ source: slug }) })
    },
  })
}
export function useForwardAll(slug: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: () => api.forwardAll(slug),
    onError,
    onSuccess: (r) => {
      toast.success(`Forwarded ${r.forwarded} deliveries`)
      qc.invalidateQueries({ queryKey: ["deliveries"] })
    },
  })
}
```

- [ ] **Step 2: Verify typecheck**

Run: `cd web/ui && npx tsc --noEmit`
Expected: no errors.

- [ ] **Step 3: Commit**

```bash
git add web/ui/src/lib/queries.ts web/ui/src/lib/utils.ts
git commit -m "feat(ui): TanStack Query hooks with cache invalidation"
```

---

### Task 7: App shell (two-pane layout) + router

**Files:**
- Create: `web/ui/src/components/app-shell.tsx`, `web/ui/src/components/list-pane.tsx`, `web/ui/src/routes/empty-state.tsx`, `web/ui/src/router.tsx`
- Modify: `web/ui/src/main.tsx`

**Interfaces:**
- Produces: `AppShell` renders the fixed two-column layout — left `<aside className="w-1/5 ...">` (section switcher `Sources | Deliveries` using `NavLink` + a `{children}` slot for the list) and right `<main className="w-4/5 ...">` containing `<Outlet/>`. `ListPane` is a generic `<nav>` with a title, a `+ New` slot, and a scrollable list of `<NavLink>` rows that highlight when active. `EmptyState({title})` centers a muted message.
- Router tree per the spec (filled in Tasks 8–13); this task wires the skeleton with placeholder route elements.

- [ ] **Step 1: Build the shell** — `web/ui/src/components/app-shell.tsx`

```tsx
import { NavLink, Outlet } from "react-router-dom"
import { cn } from "@/lib/utils"

const tab = ({ isActive }: { isActive: boolean }) =>
  cn("flex-1 py-2 text-center text-sm font-medium border-b-2",
    isActive ? "border-primary text-foreground" : "border-transparent text-muted-foreground")

export function AppShell({ list }: { list: React.ReactNode }) {
  return (
    <div className="flex h-screen w-screen overflow-hidden">
      <aside className="w-1/5 min-w-[220px] max-w-[360px] border-r flex flex-col">
        <nav className="flex border-b">
          <NavLink to="/sources" className={tab}>Sources</NavLink>
          <NavLink to="/deliveries" className={tab}>Deliveries</NavLink>
        </nav>
        <div className="flex-1 overflow-y-auto">{list}</div>
      </aside>
      <main className="flex-1 overflow-y-auto"><Outlet /></main>
    </div>
  )
}
```

- [ ] **Step 2: `ListPane`** — `web/ui/src/components/list-pane.tsx`

```tsx
import { NavLink } from "react-router-dom"
import { cn } from "@/lib/utils"

export interface ListItem { key: string; to: string; label: React.ReactNode }

export function ListPane({ header, items, empty }: {
  header?: React.ReactNode; items: ListItem[]; empty?: string
}) {
  return (
    <div className="flex flex-col">
      {header && <div className="p-2 border-b">{header}</div>}
      {items.length === 0 && <p className="p-4 text-sm text-muted-foreground">{empty ?? "Nothing yet"}</p>}
      <ul>
        {items.map((it) => (
          <li key={it.key}>
            <NavLink to={it.to} className={({ isActive }) =>
              cn("block px-4 py-2 text-sm border-b hover:bg-accent",
                 isActive && "bg-accent font-medium")}>
              {it.label}
            </NavLink>
          </li>
        ))}
      </ul>
    </div>
  )
}
```

- [ ] **Step 3: Empty state** — `web/ui/src/routes/empty-state.tsx`

```tsx
export function EmptyState({ title }: { title: string }) {
  return <div className="h-full flex items-center justify-center text-muted-foreground">{title}</div>
}
```

- [ ] **Step 4: Router skeleton** — `web/ui/src/router.tsx` (detail elements are placeholders here; later tasks replace them)

```tsx
import { createBrowserRouter, Navigate } from "react-router-dom"
import { SourcesLayout } from "./routes/sources-layout"
import { DeliveriesLayout } from "./routes/deliveries-layout"
import { EmptyState } from "./routes/empty-state"

export const router = createBrowserRouter([
  { path: "/", element: <Navigate to="/sources" replace /> },
  {
    path: "/sources", element: <SourcesLayout />,
    children: [{ index: true, element: <EmptyState title="Select a source" /> }],
  },
  {
    path: "/deliveries", element: <DeliveriesLayout />,
    children: [{ index: true, element: <EmptyState title="Select a delivery" /> }],
  },
])
```

Create minimal `web/ui/src/routes/sources-layout.tsx` and `deliveries-layout.tsx` that render `<AppShell list={<div/>} />` for now (filled in later tasks).

- [ ] **Step 5: Wire providers** — `web/ui/src/main.tsx`

```tsx
import React from "react"
import ReactDOM from "react-dom/client"
import { RouterProvider } from "react-router-dom"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { Toaster } from "sonner"
import { router } from "./router"
import "./index.css"

const qc = new QueryClient()

ReactDOM.createRoot(document.getElementById("root")!).render(
  <React.StrictMode>
    <QueryClientProvider client={qc}>
      <RouterProvider router={router} />
      <Toaster richColors />
    </QueryClientProvider>
  </React.StrictMode>,
)
```

- [ ] **Step 6: Verify it renders**

Run: `cd web/ui && npx tsc --noEmit && npm run build`
Expected: typecheck clean, build succeeds. (Optional: `npm run dev`, load `http://localhost:5173`, see the two-pane shell with Sources/Deliveries tabs.)

- [ ] **Step 7: Commit**

```bash
git add web/ui/src/components web/ui/src/routes web/ui/src/router.tsx web/ui/src/main.tsx
git commit -m "feat(ui): two-pane app shell + router skeleton"
```

---

## Phase C — Sources

### Task 8: Sources list pane + create dialog

**Files:**
- Modify: `web/ui/src/routes/sources-layout.tsx`
- Create: `web/ui/src/routes/create-source-dialog.tsx`, `web/ui/src/routes/sources-layout.test.tsx`, `web/ui/src/test/render.tsx` (test helper)

**Interfaces:**
- Consumes: `useSources`, `useCreateSource`, `ListPane`, `AppShell`.
- Produces: `SourcesLayout` renders `<AppShell list={<sources ListPane + New button>} />`. Create dialog: name (required) + mode select (`active`/`record`); on submit calls `useCreateSource` then navigates to `/sources/:slug`.

- [ ] **Step 1: Test render helper** — `web/ui/src/test/render.tsx`

```tsx
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { render } from "@testing-library/react"
import { RouterProvider, createMemoryRouter } from "react-router-dom"
import type { RouteObject } from "react-router-dom"

export function renderRoutes(routes: RouteObject[], initial = "/") {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  const router = createMemoryRouter(routes, { initialEntries: [initial] })
  return render(<QueryClientProvider client={qc}><RouterProvider router={router} /></QueryClientProvider>)
}
```

- [ ] **Step 2: Write the failing test** — `web/ui/src/routes/sources-layout.test.tsx`

```tsx
import { afterEach, beforeEach, expect, test, vi } from "vitest"
import { screen } from "@testing-library/react"
import { renderRoutes } from "@/test/render"
import { SourcesLayout } from "./sources-layout"
import { EmptyState } from "./empty-state"

beforeEach(() => {
  vi.stubGlobal("fetch", vi.fn().mockResolvedValue(
    new Response(JSON.stringify([{ id: "1", name: "GitHub", slug: "github", mode: "active",
      created_at: "", updated_at: "" }]),
      { status: 200, headers: { "Content-Type": "application/json" } })))
})
afterEach(() => vi.restoreAllMocks())

test("renders sources in the list pane", async () => {
  renderRoutes([{ path: "/sources", element: <SourcesLayout />,
    children: [{ index: true, element: <EmptyState title="Select a source" /> }] }], "/sources")
  expect(await screen.findByText("GitHub")).toBeInTheDocument()
})
```

- [ ] **Step 3: Run test to verify it fails**

Run: `cd web/ui && npx vitest run src/routes/sources-layout.test.tsx`
Expected: FAIL (list is empty placeholder).

- [ ] **Step 4: Implement `SourcesLayout`** — `web/ui/src/routes/sources-layout.tsx`

```tsx
import { AppShell } from "@/components/app-shell"
import { ListPane } from "@/components/list-pane"
import { useSources } from "@/lib/queries"
import { CreateSourceDialog } from "./create-source-dialog"

export function SourcesLayout() {
  const { data: sources = [], isLoading } = useSources()
  return (
    <AppShell list={
      <ListPane
        header={<CreateSourceDialog />}
        empty={isLoading ? "Loading…" : "No sources yet"}
        items={sources.map((s) => ({
          key: s.id, to: `/sources/${s.slug}`,
          label: (<span className="flex justify-between gap-2">
            <span className="truncate">{s.name}</span>
            <span className="text-xs text-muted-foreground">{s.mode}</span></span>),
        }))} />
    } />
  )
}
```

- [ ] **Step 5: Implement create dialog** — `web/ui/src/routes/create-source-dialog.tsx` using shadcn `Dialog`, `Input`, `Select`, `Button`, react-hook-form + zod (`z.object({ name: z.string().min(1), mode: z.enum(["active","record"]) })`); on success `navigate(\`/sources/${created.slug}\`)` and close.

- [ ] **Step 6: Run test to verify it passes**

Run: `cd web/ui && npx vitest run src/routes/sources-layout.test.tsx`
Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add web/ui/src/routes/sources-layout.tsx web/ui/src/routes/create-source-dialog.tsx web/ui/src/routes/sources-layout.test.tsx web/ui/src/test/render.tsx
git commit -m "feat(ui): sources list pane + create-source dialog"
```

---

### Task 9: Source detail shell + Overview tab

**Files:**
- Create: `web/ui/src/routes/source-detail.tsx`, `web/ui/src/routes/source-overview.tsx`
- Modify: `web/ui/src/router.tsx` (nest `/sources/:slug/:tab?` under `SourcesLayout`)

**Interfaces:**
- Produces: `SourceDetail` renders a header (name, slug) + shadcn `Tabs` whose values (`overview|actions|script|events`) are driven by the `:tab` route param via `NavLink`s, and an inner `<Outlet/>` for the tab body. `SourceOverview` shows mode with a `Switch` (calls `useUpdateSource` to toggle `active`/`record`), the ingest URL `\`${VITE_API_BASE_URL}/webhooks/${slug}\`` with a copy button, and created/updated timestamps.
- Consumes: `useSource`, `useUpdateSource`.

- [ ] **Step 1: Implement `SourceDetail`** with tab nav routing to `/sources/:slug/<tab>` and default redirect to `overview`.

- [ ] **Step 2: Implement `SourceOverview`** (mode Switch, copyable ingest URL, timestamps).

- [ ] **Step 3: Update router** — nest under the `/sources` route:

```tsx
{
  path: ":slug", element: <SourceDetail />,
  children: [
    { index: true, element: <Navigate to="overview" replace /> },
    { path: "overview", element: <SourceOverview /> },
    { path: "actions", element: <SourceActions /> },   // Task 10
    { path: "script", element: <SourceScript /> },      // Task 11
    { path: "events", element: <SourceEvents /> },      // Task 12
  ],
}
```

Stub `SourceActions/SourceScript/SourceEvents` as `() => null` until their tasks land so the build stays green.

- [ ] **Step 4: Write a test** — `source-overview.test.tsx`: mock `getSource` → render at `/sources/github/overview`, assert the ingest URL text and mode label appear.

- [ ] **Step 5: Run tests**

Run: `cd web/ui && npx vitest run src/routes/source-overview.test.tsx && npx tsc --noEmit`
Expected: PASS, typecheck clean.

- [ ] **Step 6: Commit**

```bash
git add web/ui/src/routes/source-detail.tsx web/ui/src/routes/source-overview.tsx web/ui/src/routes/source-overview.test.tsx web/ui/src/router.tsx
git commit -m "feat(ui): source detail shell + overview tab"
```

---

### Task 10: Actions tab + action-form (5 types)

**Files:**
- Create: `web/ui/src/routes/source-actions.tsx`, `web/ui/src/routes/action-form.tsx`, `web/ui/src/routes/action-form.test.tsx`
- Create: `web/ui/src/components/action-type-badge.tsx`

**Interfaces:**
- Produces: `SourceActions` lists actions (type badge, target/summary, active `Switch` via `useUpdateAction`, edit + delete). `ActionForm` is a dialog whose fields switch on a `type` `Select` driven by a zod discriminated union; submit maps to `createAction`/`updateAction` payload. `ActionTypeBadge({type})` → colored `Badge`.
- Consumes: `useActions`, `useCreateAction`, `useUpdateAction`, `useDeleteAction`.
- Zod schema (exact field names from Global Constraints):

```ts
import { z } from "zod"
const webhook = z.object({ type: z.literal("webhook"), target_url: z.string().url(),
  signing_secret: z.string().optional(), transform_script: z.string().optional() })
const javascript = z.object({ type: z.literal("javascript"), script_body: z.string().min(1) })
const slack = z.object({ type: z.literal("slack"),
  config: z.object({ webhook_url: z.string().url(), channel: z.string().optional(), username: z.string().optional() }) })
const smtp = z.object({ type: z.literal("smtp"),
  config: z.object({ host: z.string(), port: z.coerce.number(), username: z.string(),
    password: z.string(), from: z.string(), to: z.string(), subject: z.string() }) })
const twilio = z.object({ type: z.literal("twilio"),
  config: z.object({ account_sid: z.string(), auth_token: z.string(), from: z.string(),
    to: z.string(), body_template: z.string().optional() }) })
export const actionSchema = z.discriminatedUnion("type", [webhook, javascript, slack, smtp, twilio])
export type ActionFormValues = z.infer<typeof actionSchema>
```

- [ ] **Step 1: Write the failing test** — `action-form.test.tsx`: render `ActionForm` (create mode, no network needed for field rendering), open it, select type `slack`, assert a `webhook_url` field appears and `target_url` does not; select `webhook`, assert `target_url` appears.

- [ ] **Step 2: Run it — expect FAIL** (`ActionForm` undefined).

Run: `cd web/ui && npx vitest run src/routes/action-form.test.tsx`

- [ ] **Step 3: Implement `ActionTypeBadge`, `ActionForm`, `SourceActions`.** `ActionForm` uses `useForm({ resolver: zodResolver(actionSchema) })`, watches `type`, and renders the field group for the selected type. On submit, POST/PATCH via the mutation, then close.

- [ ] **Step 4: Run test — expect PASS.**

- [ ] **Step 5: Manual smoke (optional):** with API + `npm run dev` running, create a webhook action on a source and confirm it appears.

- [ ] **Step 6: Commit**

```bash
git add web/ui/src/routes/source-actions.tsx web/ui/src/routes/action-form.tsx web/ui/src/routes/action-form.test.tsx web/ui/src/components/action-type-badge.tsx
git commit -m "feat(ui): actions tab + type-aware action form"
```

---

### Task 11: Script tab (CodeMirror + test-run)

**Files:**
- Create: `web/ui/src/routes/source-script.tsx`, `web/ui/src/components/script-editor.tsx`, `web/ui/src/components/json-viewer.tsx`
- Create: `web/ui/src/components/json-viewer.test.tsx`

**Interfaces:**
- Produces: `ScriptEditor({value,onChange,language})` wraps `@uiw/react-codemirror` with `javascript()`/`json()` extensions. `SourceScript` shows the editor seeded from `source.script_body`, a Save button (`useUpdateSource({script_body})`), a Clear button (`useUpdateSource({script_body:""})`), and a test panel: a delivery picker (from `useDeliveries({source: slug})`) + "Run" → `testScript(slug,{script_body, delivery_id})`, rendering `{result}` via `JsonViewer` or the `error` string. `JsonViewer({data})` pretty-prints JSON in a `<pre>` with wrapping/scroll.
- Consumes: `useSource`, `useUpdateSource`, `useDeliveries`, `api.testScript`.

- [ ] **Step 1: Write the failing test** — `json-viewer.test.tsx`: render `<JsonViewer data={{a:1}} />`, assert the text `"a": 1` (formatted) appears.

- [ ] **Step 2: Run it — expect FAIL.**

- [ ] **Step 3: Implement `JsonViewer`, `ScriptEditor`, `SourceScript`.**

- [ ] **Step 4: Run `json-viewer.test.tsx` — expect PASS; `npx tsc --noEmit` clean.**

- [ ] **Step 5: Commit**

```bash
git add web/ui/src/routes/source-script.tsx web/ui/src/components/script-editor.tsx web/ui/src/components/json-viewer.tsx web/ui/src/components/json-viewer.test.tsx
git commit -m "feat(ui): script tab with CodeMirror editor and test-run"
```

---

### Task 12: Events tab (per-source deliveries + forward)

**Files:**
- Create: `web/ui/src/routes/source-events.tsx`
- Create: `web/ui/src/components/status-badge.tsx`

**Interfaces:**
- Produces: `SourceEvents` uses `useDeliveries({source: slug})` to render a `Table` (received_at, status via `StatusBadge`, idempotency_key, link to `/deliveries/:id`). For `recorded` rows show a per-row **Forward** button (`useForwardDelivery(slug)`); a **Forward all** button (`useForwardAll(slug)`) in the header. `StatusBadge({status})` maps delivery/attempt statuses → colored `Badge` (completed/success → green, failed → red, pending/processing → amber, recorded → slate).
- Consumes: `useDeliveries`, `useForwardDelivery`, `useForwardAll`.

- [ ] **Step 1: Write the failing test** — `status-badge.test.tsx`: assert `StatusBadge status="completed"` renders text "completed" and has the success class; `status="failed"` has the destructive class.

- [ ] **Step 2: Run it — expect FAIL.**

- [ ] **Step 3: Implement `StatusBadge` and `SourceEvents`.**

- [ ] **Step 4: Run test — expect PASS.**

- [ ] **Step 5: Commit**

```bash
git add web/ui/src/routes/source-events.tsx web/ui/src/components/status-badge.tsx web/ui/src/components/status-badge.test.tsx
git commit -m "feat(ui): source events tab with forward actions"
```

---

## Phase D — Deliveries

### Task 13: Deliveries list pane + delivery detail

**Files:**
- Modify: `web/ui/src/routes/deliveries-layout.tsx`
- Create: `web/ui/src/routes/delivery-detail.tsx`, `web/ui/src/routes/delivery-detail.test.tsx`
- Modify: `web/ui/src/router.tsx` (nest `:id` under `/deliveries`)

**Interfaces:**
- Produces: `DeliveriesLayout` renders `<AppShell list={<deliveries ListPane>} />` from `useDeliveries({})` (row: short id + `StatusBadge`), with a status filter `Select` (all/pending/processing/completed/failed/recorded) filtering client-side. `DeliveryDetail` shows status, source, received_at, `JsonViewer` for headers + payload (+ transformed_* when present), a **Forward** button for `recorded`, and an attempts timeline from `useAttempts(id)` (attempt_number, `StatusBadge`, response_status, error_message, next_retry_at).
- Consumes: `useDeliveries`, `useDelivery`, `useAttempts`, `useForwardDelivery`, `JsonViewer`, `StatusBadge`.

- [ ] **Step 1: Write the failing test** — `delivery-detail.test.tsx`: mock `getDelivery` + `listAttempts`, render at `/deliveries/abc`, assert payload JSON and one attempt row appear.

- [ ] **Step 2: Run it — expect FAIL.**

- [ ] **Step 3: Implement `DeliveriesLayout` + `DeliveryDetail`; update router:**

```tsx
{
  path: "/deliveries", element: <DeliveriesLayout />,
  children: [
    { index: true, element: <EmptyState title="Select a delivery" /> },
    { path: ":id", element: <DeliveryDetail /> },
  ],
}
```

- [ ] **Step 4: Run test — expect PASS; `npx tsc --noEmit` clean.**

- [ ] **Step 5: Commit**

```bash
git add web/ui/src/routes/deliveries-layout.tsx web/ui/src/routes/delivery-detail.tsx web/ui/src/routes/delivery-detail.test.tsx web/ui/src/router.tsx
git commit -m "feat(ui): deliveries list + delivery detail with attempts"
```

---

## Phase E — Polish & verification

### Task 14: Error boundaries, README, end-to-end smoke

**Files:**
- Create: `web/ui/src/routes/error-boundary.tsx`, `web/ui/README.md`
- Modify: `web/ui/src/router.tsx` (attach `errorElement`), `README.md` (root — link the SPA)

**Interfaces:**
- Produces: `RouteError` component reading `useRouteError()` → friendly message; attached as `errorElement` on the `/sources` and `/deliveries` routes. `web/ui/README.md` documents dev setup (`npm install`, `.env`, `npm run dev`, `npm run test`, `npm run build`), the `VITE_API_BASE_URL` var, and that the API must run with `CORS_ALLOWED_ORIGINS` including the dev origin.

- [ ] **Step 1: Implement `RouteError`, attach `errorElement`.**

- [ ] **Step 2: Write `web/ui/README.md` and add a "React admin UI" link to the root `README.md`.**

- [ ] **Step 3: Full test + build gate**

Run: `cd web/ui && npm run test && npx tsc --noEmit && npm run build`
Expected: all tests PASS, typecheck clean, production build succeeds.

- [ ] **Step 4: Backend gate**

Run (repo root): `go build ./... && go test ./...`
Expected: build + unit tests PASS.

- [ ] **Step 5: End-to-end smoke (manual)**

Start supporting services + API (`make docker-up-supporting-svc`, `make migrate-up`, `make run-api` with `CORS_ALLOWED_ORIGINS=http://localhost:5173`), start `make ui-dev`, then in the browser: create a source, add a webhook action, POST a test webhook to `/webhooks/:slug`, and confirm the delivery + attempt appear in the Events/Deliveries views. Note any gaps.

- [ ] **Step 6: Commit**

```bash
git add web/ui/src/routes/error-boundary.tsx web/ui/README.md web/ui/src/router.tsx README.md
git commit -m "feat(ui): error boundaries + docs; finalize SPA"
```

---

## Self-review notes (coverage map)

- Spec "backend delta (CORS + 3 endpoints)" → Tasks 1–3.
- Spec "two-pane layout / section switcher" → Task 7 (`AppShell`).
- Spec routes (`/sources`, `/sources/:slug/:tab`, `/deliveries`, `/deliveries/:id`) → Tasks 7, 9, 13.
- Spec source-detail tabs (Overview/Actions/Script/Events) → Tasks 9–12.
- Spec 5 action types (exact config fields) → Task 10.
- Spec script test-run → Tasks 2 + 11.
- Spec record-mode forward / forward-all → Tasks 3 + 12 + 13.
- Spec delivery detail + attempts timeline → Task 13.
- Spec data flow (apiFetch/ApiError, query keys, invalidation, toasts) → Tasks 5–6.
- Spec testing (Vitest + RTL + MSW-or-fetch-mock) → per-task tests + Task 14 gate. (Fetch-mocking via `vi.stubGlobal` is used instead of full MSW where simpler; MSW remains available and installed for richer route tests.)
- Spec "Go-template UI untouched" → no task modifies `web/*.go` behavior except adding new handlers in `internal/handler`.
