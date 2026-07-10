# NitroHook React Admin SPA — Design

**Date:** 2026-07-10
**Status:** Approved (design), pending implementation plan

## Goal

Build a modern single-page admin frontend for NitroHook using **Vite + React + TypeScript + React Router v7 + shadcn/ui**. It is a **full replacement** for the existing server-rendered Go-template UI (`web/templates/`), consuming the existing `/api` JSON endpoints. It lives in `web/ui/` inside this repo and is served **standalone** (its own Vite build / static host), talking to the Go API over HTTP. **No authentication** for now — matches the current API.

The Go-template UI stays untouched until the user chooses to retire it.

## Decisions (from brainstorming)

| Decision | Choice |
|---|---|
| Relationship to Go UI | Full replacement (all features, all 5 action types, script editor, deliveries + attempts) |
| Location / serving | `web/ui/` subdir in repo, deployed/served standalone |
| Auth | None for now; add CORS so the SPA can call the API cross-origin |
| Layout | Two-pane master–detail: ~20% list pane (left), ~80% detail pane (right) |

## Architecture

The SPA is a pure client of the existing `/api` JSON API. Data flow:

```
React SPA (Vite dev :5173 / static host)  ──HTTP──▶  Go API (/api, :8080)  ──▶  Postgres / Redis
```

### Backend delta (Go)

The existing `/api` covers sources CRUD, actions CRUD (PATCH covers active-toggle), and deliveries
(list with `?source=slug&limit=N`, get, attempts). Three HTML-only features have no JSON equivalent and
must be added for full parity, plus CORS:

1. **CORS middleware** on the API. Allow the SPA origin, configurable via env
   (e.g. `CORS_ALLOWED_ORIGINS`, default `http://localhost:5173`). Handles preflight for
   `PATCH`/`DELETE`/`POST` + `Content-Type`/`Authorization` headers.
2. **`POST /api/sources/:slug/script/test`** — JSON version of `web.TestSourceScript`.
   Request `{ script_body: string, delivery_id: string }`. Response
   `{ payload: json, headers: json, actions: [...], logs: string[], error?: string }`.
   Reuses the existing `script` package execution path; only the HTML-fragment rendering is replaced with JSON.
3. **`POST /api/deliveries/:id/forward`** — JSON version of `web.ForwardDelivery`. Re-publishes a
   recorded delivery to the Redis stream. Response `{ status: "forwarded" }` or error.
4. **`POST /api/sources/:slug/deliveries/forward-all`** — JSON version of `web.ForwardAllRecorded`.
   Batch-forwards all recorded deliveries for a source. Response `{ forwarded: number }`.

These reuse existing store/script/worker-publish logic; they are thin JSON adapters over code that
already exists in the `web` package, extracted/shared as needed so logic is not duplicated.

## Frontend stack

| Concern | Choice |
|---|---|
| Build | Vite + React 18 + TypeScript |
| Routing | React Router v7, `createBrowserRouter` with nested data routes |
| Components | shadcn/ui (Radix primitives + Tailwind CSS) |
| Server state | TanStack Query over a thin typed `fetch` client |
| Forms + validation | react-hook-form + zod (discriminated union for the 5 action types) |
| JS script editor | CodeMirror 6 via `@uiw/react-codemirror` (javascript + json language support) |
| Toasts | sonner (shadcn integration) |
| Icons | lucide-react |

Rationale: the UI is almost entirely CRUD-over-REST with lists that need refetch/invalidation
(create action → refetch that source's actions; forward delivery → refetch deliveries). TanStack Query
removes manual loading/error/caching boilerplate. CodeMirror is lighter than Monaco for the script editor.

## Layout — two-pane master–detail

A persistent two-column shell. The list pane never unmounts when navigating into a detail; only the
detail pane (an `<Outlet/>`) swaps.

```
┌────────────────────────┬──────────────────────────────────────────────┐
│  LIST PANE  (~20%)      │  DETAIL PANE  (~80%)  = <Outlet/>            │
│  ┌──────────────────┐   │                                              │
│  │ [Sources][Deliv.] │  │   (selected item detail, or empty state)     │
│  ├──────────────────┤   │                                              │
│  │ • payments        │  │                                              │
│  │ • github  (active)│  │                                              │
│  │ • stripe          │  │                                              │
│  │ …scrollable list  │  │                                              │
│  └──────────────────┘   │                                              │
│  [+ New]                │                                              │
└────────────────────────┴──────────────────────────────────────────────┘
```

- Top of the list pane: a **section switcher** (Sources | Deliveries) toggling which list is shown.
- List pane is independently scrollable; selected row is highlighted.
- On narrow viewports (< md), the panes collapse to single-column: list is the page, tapping an item
  navigates to a full-screen detail with a back affordance. (Responsive, not a separate design.)

## Routes

```
/                         → redirect to /sources
/sources                  → shell + sources list; detail pane = "Select a source" empty state
/sources/:slug            → shell + sources list (row highlighted); detail = source detail w/ tabs
/sources/:slug/:tab       → Overview | Actions | Script | Events   (tab = nested route segment; default overview)
/deliveries               → shell + deliveries list; detail = "Select a delivery" empty state
/deliveries/:id           → shell + deliveries list; detail = delivery detail (payload/headers + attempts)
```

Source-detail tabs:
- **Overview** — name, slug, mode (active/record) with mode toggle, webhook ingest URL, created/updated.
- **Actions** — list of actions with type badges + active toggle; create/edit via `action-form` dialog.
- **Script** — CodeMirror editor for the source transform script; "test against a recorded delivery"
  panel calling `POST /api/sources/:slug/script/test`; save / clear.
- **Events** — deliveries filtered to this source (`/api/deliveries?source=:slug`); for record-mode
  sources, per-row **Forward** and a **Forward all** button.

## File structure

```
web/ui/
  index.html
  package.json  vite.config.ts  tsconfig.json  tailwind.config.ts  components.json
  .env.example                      # VITE_API_BASE_URL
  src/
    main.tsx                        # QueryClientProvider + RouterProvider
    router.tsx                      # createBrowserRouter route tree
    lib/
      api.ts                        # apiFetch wrapper + ApiError; endpoint fns
      types.ts                      # TS mirrors of Go models
      queries.ts                    # query keys + useQuery/useMutation hooks
      utils.ts                      # cn() etc.
    components/
      ui/                           # shadcn primitives (button, dialog, table, tabs, badge, …)
      app-shell.tsx                 # two-pane layout + section switcher
      list-pane.tsx                 # generic scrollable selectable list
      status-badge.tsx              # delivery/attempt status → colored badge
      action-type-badge.tsx        # action type → badge/icon
      json-viewer.tsx               # collapsible syntax-highlighted JSON
      script-editor.tsx             # CodeMirror wrapper
      confirm-dialog.tsx
    routes/
      sources-layout.tsx            # list pane = sources; <Outlet/>
      source-detail.tsx             # tabs
      source-overview.tsx  source-actions.tsx  source-script.tsx  source-events.tsx
      action-form.tsx               # create/edit dialog; fields switch by 5 types (zod union)
      deliveries-layout.tsx         # list pane = deliveries; <Outlet/>
      delivery-detail.tsx
      empty-state.tsx
```

## Data flow & error handling

- **`apiFetch(path, opts)`** — prepends `VITE_API_BASE_URL`, sets JSON headers, and on non-2xx throws a
  typed `ApiError { status, message }`. The Go API returns plain-text error bodies, so the wrapper reads
  the body as the message.
- **Query keys** mirror routes: `['sources']`, `['source', slug]`, `['source', slug, 'actions']`,
  `['deliveries', { source, status }]`, `['delivery', id, 'attempts']`. Mutations invalidate the
  affected keys (e.g. creating an action invalidates `['source', slug, 'actions']`).
- **Errors**: each route has an error boundary; mutation errors surface via sonner toasts using
  `ApiError.message`. Lists show skeleton loaders and explicit empty states.

## The 5 action types (form model)

A zod **discriminated union on `type`** drives `action-form`:

| Type | Primary fields |
|---|---|
| `webhook` | `target_url` (required), `signing_secret` (optional), `transform_script` (optional) |
| `slack` | `config.webhook_url` / channel (per existing Slack dispatcher config) |
| `smtp` | `config` (to/subject/etc. per SMTP dispatcher) |
| `twilio` | `config` (to/from per Twilio dispatcher) |
| `javascript` | `script_body` (CodeMirror editor) |

Exact `config` field names are read from the corresponding `dispatch` package structs during
implementation so the form matches what the dispatcher expects.

## Testing

- **Vitest + React Testing Library** — component tests: `action-form` type switching, `status-badge`
  mapping, `json-viewer` collapse.
- **MSW (Mock Service Worker)** — mock `/api` for route/integration tests; no live backend in CI.
- **TypeScript** — `types.ts` mirrors Go structs; first line of defense against API drift.
- Backend: Go unit/integration tests for the 3 new JSON endpoints, following existing
  `handler_integration_test.go` patterns.

## Out of scope (YAGNI)

- Authentication / user management (explicitly deferred).
- Embedding the SPA build into the Go binary (standalone serving chosen).
- Retiring/deleting the Go-template UI (kept as-is during transition).
- Realtime/websocket delivery streaming (poll/refetch via TanStack Query is sufficient).
