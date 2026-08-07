# Frontend Query Error Handling Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Surface failed data-fetching (React Query reads) inline with a Retry button instead of rendering misleading empty states.

**Architecture:** A shared `ErrorState` component + `describeApiError` helper, applied per-view via each query's `isError`/`error`/`refetch`. QueryClient gets a retry predicate that skips 4xx. No backend changes.

**Tech Stack:** React 19, TanStack React Query v5, React Router v7, Tailwind/shadcn, Vitest + Testing Library.

**Spec:** `docs/superpowers/specs/2026-07-13-frontend-query-error-handling-design.md`

## Global Constraints

- All frontend work lives under `web/ui/`. Run commands from `web/ui/` (e.g. `cd web/ui && npx vitest run <file>`).
- Tests are colocated with source (`foo.test.tsx` next to `foo.tsx`) and use `renderRoutes` from `src/test/render.tsx` plus `vi.stubGlobal("fetch", ...)` — follow `src/routes/dashboard.test.tsx` as the pattern.
- Error copy convention: panel headings are "Couldn't load <things>" (sources, deliveries, actions, attempts, delivery).
- Do not change mutation error handling (already toasts) or the route-level `RouteError` boundary.
- `CHANGELOG.md` update is folded into the final task.

---

### Task 1: `describeApiError` helper

**Files:**
- Modify: `web/ui/src/lib/utils.ts`
- Test: `web/ui/src/lib/utils.test.ts` (create)

**Interfaces:**
- Consumes: `ApiError` from `@/lib/api` (exists: `class ApiError extends Error { status: number }`).
- Produces: `describeApiError(err: unknown): string` — used by Task 2's `ErrorState`.

- [ ] **Step 1: Write the failing test**

Create `web/ui/src/lib/utils.test.ts`:

```ts
import { expect, test } from "vitest"
import { ApiError } from "./api"
import { describeApiError } from "./utils"

test("describeApiError returns the ApiError message", () => {
  expect(describeApiError(new ApiError(500, "internal error"))).toBe("internal error")
})

test("describeApiError maps network TypeError to a reachability message", () => {
  expect(describeApiError(new TypeError("Failed to fetch"))).toBe(
    "Can't reach the API. Check that the server is running.",
  )
})

test("describeApiError falls back for unknown values", () => {
  expect(describeApiError(undefined)).toBe("An unexpected error occurred.")
  expect(describeApiError("boom")).toBe("An unexpected error occurred.")
})
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd web/ui && npx vitest run src/lib/utils.test.ts`
Expected: FAIL — `describeApiError` is not exported.

- [ ] **Step 3: Write minimal implementation**

Append to `web/ui/src/lib/utils.ts` (add the import at the top of the file):

```ts
import { ApiError } from "./api"

export function describeApiError(err: unknown): string {
  if (err instanceof ApiError) return err.message
  if (err instanceof TypeError) return "Can't reach the API. Check that the server is running."
  return "An unexpected error occurred."
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd web/ui && npx vitest run src/lib/utils.test.ts`
Expected: PASS (3 tests)

- [ ] **Step 5: Commit**

```bash
git add web/ui/src/lib/utils.ts web/ui/src/lib/utils.test.ts
git commit -m "feat(ui): add describeApiError helper for human-readable fetch errors"
```

---

### Task 2: `ErrorState` component

**Files:**
- Create: `web/ui/src/components/error-state.tsx`
- Test: `web/ui/src/components/error-state.test.tsx`

**Interfaces:**
- Consumes: `describeApiError` (Task 1), `Button` from `@/components/ui/button`.
- Produces: `ErrorState({ title, error, onRetry }: { title: string; error: unknown; onRetry?: () => void })` — used by Tasks 4–7.

- [ ] **Step 1: Write the failing test**

Create `web/ui/src/components/error-state.test.tsx`:

```tsx
import { expect, test, vi } from "vitest"
import { render, screen } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { ApiError } from "@/lib/api"
import { ErrorState } from "./error-state"

test("renders title, error detail, and calls onRetry", async () => {
  const onRetry = vi.fn()
  render(<ErrorState title="Couldn't load sources" error={new ApiError(500, "internal error")} onRetry={onRetry} />)
  expect(screen.getByText("Couldn't load sources")).toBeInTheDocument()
  expect(screen.getByText("internal error")).toBeInTheDocument()
  await userEvent.click(screen.getByRole("button", { name: "Retry" }))
  expect(onRetry).toHaveBeenCalledOnce()
})

test("omits the retry button when onRetry is not given", () => {
  render(<ErrorState title="Couldn't load sources" error={new TypeError("Failed to fetch")} />)
  expect(screen.queryByRole("button", { name: "Retry" })).not.toBeInTheDocument()
})
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd web/ui && npx vitest run src/components/error-state.test.tsx`
Expected: FAIL — module `./error-state` not found.

- [ ] **Step 3: Write minimal implementation**

Create `web/ui/src/components/error-state.tsx`:

```tsx
import { Button } from "@/components/ui/button"
import { describeApiError } from "@/lib/utils"

export function ErrorState({ title, error, onRetry }: {
  title: string; error: unknown; onRetry?: () => void
}) {
  return (
    <div className="flex h-full flex-col items-center justify-center gap-2 p-4 text-center">
      <p className="text-sm font-medium">{title}</p>
      <p className="text-sm text-muted-foreground">{describeApiError(error)}</p>
      {onRetry && (
        <Button type="button" variant="outline" size="sm" onClick={onRetry}>
          Retry
        </Button>
      )}
    </div>
  )
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd web/ui && npx vitest run src/components/error-state.test.tsx`
Expected: PASS (2 tests)

- [ ] **Step 5: Commit**

```bash
git add web/ui/src/components/error-state.tsx web/ui/src/components/error-state.test.tsx
git commit -m "feat(ui): add ErrorState component with retry"
```

---

### Task 3: QueryClient retry predicate (skip 4xx)

**Files:**
- Modify: `web/ui/src/lib/api.ts`
- Modify: `web/ui/src/main.tsx`
- Test: `web/ui/src/lib/api.test.ts` (append)

**Interfaces:**
- Produces: `shouldRetryQuery(failureCount: number, error: unknown): boolean` exported from `@/lib/api`.

- [ ] **Step 1: Write the failing test**

Append to `web/ui/src/lib/api.test.ts`:

```ts
import { ApiError, shouldRetryQuery } from "./api"

test("shouldRetryQuery skips 4xx errors", () => {
  expect(shouldRetryQuery(0, new ApiError(404, "not found"))).toBe(false)
  expect(shouldRetryQuery(0, new ApiError(400, "bad request"))).toBe(false)
})

test("shouldRetryQuery retries 5xx and network errors up to 2 times", () => {
  expect(shouldRetryQuery(0, new ApiError(500, "boom"))).toBe(true)
  expect(shouldRetryQuery(1, new TypeError("Failed to fetch"))).toBe(true)
  expect(shouldRetryQuery(2, new ApiError(500, "boom"))).toBe(false)
})
```

(Merge imports with the file's existing imports — it already imports from `./api` and `vitest`.)

- [ ] **Step 2: Run test to verify it fails**

Run: `cd web/ui && npx vitest run src/lib/api.test.ts`
Expected: FAIL — `shouldRetryQuery` is not exported.

- [ ] **Step 3: Write minimal implementation**

Append to `web/ui/src/lib/api.ts`:

```ts
// Query retry policy: 4xx responses are deterministic (retrying a 404 three
// times just delays the error UI), so only network failures and 5xx retry.
export function shouldRetryQuery(failureCount: number, error: unknown): boolean {
  if (error instanceof ApiError && error.status < 500) return false
  return failureCount < 2
}
```

In `web/ui/src/main.tsx`, change:

```ts
const qc = new QueryClient()
```

to:

```ts
import { shouldRetryQuery } from '@/lib/api'

const qc = new QueryClient({
  defaultOptions: { queries: { retry: shouldRetryQuery } },
})
```

(Place the import with the other imports at the top.)

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd web/ui && npx vitest run src/lib/api.test.ts`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add web/ui/src/lib/api.ts web/ui/src/lib/api.test.ts web/ui/src/main.tsx
git commit -m "feat(ui): don't retry 4xx query errors; cap retries at 2"
```

---

### Task 4: List layouts — `ListPane` error slot, `SourcesLayout`, `DeliveriesLayout`

**Files:**
- Modify: `web/ui/src/components/list-pane.tsx`
- Modify: `web/ui/src/routes/sources-layout.tsx`
- Modify: `web/ui/src/routes/deliveries-layout.tsx`
- Test: `web/ui/src/routes/sources-layout.test.tsx` (append)
- Test: `web/ui/src/routes/deliveries-layout.test.tsx` (append — the file exists with deep-link filter tests; reuse its fetch-stub helpers)

**Interfaces:**
- Consumes: `ErrorState` (Task 2).
- Produces: `ListPane` gains optional `error?: React.ReactNode`; when set, it renders in place of the empty message and item list (header still renders).

- [ ] **Step 1: Write the failing tests**

Append to `web/ui/src/routes/sources-layout.test.tsx` (reuse its existing fetch-stubbing helpers/imports; add `userEvent` import if absent):

```tsx
test("shows error panel instead of 'No sources yet' when fetch fails, and retries", async () => {
  const fetchMock = vi.fn(() => Promise.reject(new TypeError("Failed to fetch")))
  vi.stubGlobal("fetch", fetchMock)
  renderRoutes([{ path: "/sources", element: <SourcesLayout /> }], "/sources")

  expect(await screen.findByText("Couldn't load sources")).toBeInTheDocument()
  expect(screen.queryByText("No sources yet")).not.toBeInTheDocument()

  fetchMock.mockImplementation(() =>
    Promise.resolve(new Response(JSON.stringify(
      [{ id: "s1", name: "GitHub", slug: "github", mode: "active", created_at: "", updated_at: "" }],
    ), { status: 200, headers: { "Content-Type": "application/json" } })),
  )
  await userEvent.click(screen.getByRole("button", { name: "Retry" }))
  expect(await screen.findByText("GitHub")).toBeInTheDocument()
})
```

Append to `web/ui/src/routes/deliveries-layout.test.tsx` (it already imports `vi`, `screen`, `renderRoutes`, `DeliveriesLayout`, and defines a `routes` array — reuse them; the per-test `vi.stubGlobal` below overrides the file's `beforeEach` stub):

```tsx
test("shows error panel instead of 'No deliveries yet' when fetch fails", async () => {
  vi.stubGlobal("fetch", vi.fn(() =>
    Promise.resolve(new Response("internal error", { status: 500 })),
  ))
  renderRoutes(routes, "/deliveries")

  expect(await screen.findByText("Couldn't load deliveries")).toBeInTheDocument()
  expect(screen.getByText("internal error")).toBeInTheDocument()
  expect(screen.queryByText("No deliveries yet")).not.toBeInTheDocument()
})
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd web/ui && npx vitest run src/routes/sources-layout.test.tsx src/routes/deliveries-layout.test.tsx`
Expected: FAIL — error panel text not found (layouts render empty states today).

- [ ] **Step 3: Implement**

`web/ui/src/components/list-pane.tsx` — add the `error` prop:

```tsx
export function ListPane({ header, items, empty, error }: {
  header?: React.ReactNode; items: ListItem[]; empty?: string; error?: React.ReactNode
}) {
  return (
    <div className="flex flex-col">
      {header && <div className="p-2 border-b">{header}</div>}
      {error ?? (
        <>
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
        </>
      )}
    </div>
  )
}
```

`web/ui/src/routes/sources-layout.tsx` — destructure error state and pass the slot:

```tsx
import { ErrorState } from "@/components/error-state"
// ...
const { data: sources = [], isLoading, isError, error, refetch } = useSources()
// ...
<ListPane
  header={/* unchanged */}
  error={isError
    ? <ErrorState title="Couldn't load sources" error={error} onRetry={() => refetch()} />
    : undefined}
  empty={isLoading ? "Loading…" : query ? "No matching sources" : "No sources yet"}
  items={/* unchanged */}
/>
```

`web/ui/src/routes/deliveries-layout.tsx` — same pattern:

```tsx
import { ErrorState } from "@/components/error-state"
// ...
const { data: deliveries = [], isLoading, isError, error, refetch } = useDeliveries({})
// ...
<ListPane
  header={/* unchanged */}
  error={isError
    ? <ErrorState title="Couldn't load deliveries" error={error} onRetry={() => refetch()} />
    : undefined}
  empty={isLoading ? "Loading…" : query ? "No matching deliveries" : "No deliveries yet"}
  items={/* unchanged */}
/>
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd web/ui && npx vitest run src/routes/sources-layout.test.tsx src/routes/deliveries-layout.test.tsx src/components`
Expected: PASS (including pre-existing layout tests)

- [ ] **Step 5: Commit**

```bash
git add web/ui/src/components/list-pane.tsx web/ui/src/routes/sources-layout.tsx web/ui/src/routes/deliveries-layout.tsx web/ui/src/routes/sources-layout.test.tsx web/ui/src/routes/deliveries-layout.test.tsx
git commit -m "feat(ui): show inline errors with retry in source/delivery list panes"
```

---

### Task 5: Dashboard error handling

**Files:**
- Modify: `web/ui/src/routes/dashboard.tsx`
- Test: `web/ui/src/routes/dashboard.test.tsx` (append)

**Interfaces:**
- Consumes: `ErrorState` (Task 2).
- Behavior: when a backing query errors, stat tiles show `"—"` (not fake zeros), and the affected card bodies (Activity + Recent deliveries for the deliveries query; Sources card for the sources query) render `ErrorState`.

- [ ] **Step 1: Write the failing test**

Append to `web/ui/src/routes/dashboard.test.tsx`:

```tsx
test("shows error panels and dashes instead of fake zeros when the API is unreachable", async () => {
  vi.stubGlobal("fetch", vi.fn(() => Promise.reject(new TypeError("Failed to fetch"))))
  renderRoutes([{ path: "/", element: <Dashboard /> }], "/")

  // Deliveries panel appears in both the Activity and Recent-deliveries cards.
  expect((await screen.findAllByText("Couldn't load deliveries")).length).toBeGreaterThan(0)
  expect(screen.getByText("Couldn't load sources")).toBeInTheDocument()
  expect(screen.queryByText(/No deliveries yet/)).not.toBeInTheDocument()
  expect(screen.queryByText("Create a source")).not.toBeInTheDocument()
  // All four stat tiles degrade to em-dashes rather than showing 0.
  expect(screen.getAllByText("—").length).toBeGreaterThanOrEqual(4)
})
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd web/ui && npx vitest run src/routes/dashboard.test.tsx`
Expected: FAIL — dashboard currently renders empty-state copy and `0` tiles.

- [ ] **Step 3: Implement**

In `web/ui/src/routes/dashboard.tsx`:

Destructure error state (top of `Dashboard`):

```tsx
const {
  data: sources = [], isLoading: sourcesLoading,
  isError: sourcesError, error: sourcesErr, refetch: refetchSources,
} = useSources()
const {
  data: deliveries = [], isLoading: deliveriesLoading,
  isError: deliveriesError, error: deliveriesErr, refetch: refetchDeliveries,
} = useDeliveries({ limit: WINDOW_LIMIT }, { refetchInterval: 10_000 })
const loading = sourcesLoading || deliveriesLoading
```

Stat tiles — degrade to `"—"` when the backing query errored:

```tsx
<StatTile
  label="Sources"
  value={sourcesError ? "—" : loading ? null : String(sources.length)}
  hint={sourcesError ? "unavailable" : `${activeSources} active · ${sources.length - activeSources} recording`}
/>
<StatTile
  label="Deliveries · 24h"
  value={deliveriesError ? "—" : loading ? null : String(last24h.length)}
  hint={deliveriesError ? "unavailable" : "received in the last day"}
/>
<StatTile
  label="Failed"
  value={deliveriesError ? "—" : loading ? null : String(failed)}
  hint={deliveriesError ? "unavailable" : failed > 0 ? "needs attention" : "all clear"}
  alert={!deliveriesError && failed > 0}
/>
<StatTile
  label="Success rate"
  value={deliveriesError ? "—" : loading ? null : successRate === null ? "—" : `${successRate}%`}
  hint={deliveriesError ? "unavailable" : terminal === 0 ? "no completed deliveries yet" : `of ${terminal} finished deliveries`}
/>
```

Activity card body — first branch:

```tsx
<CardContent className="flex flex-col gap-4">
  {deliveriesError ? (
    <ErrorState title="Couldn't load deliveries" error={deliveriesErr} onRetry={() => refetchDeliveries()} />
  ) : (
    <>
      <HourlyBars deliveries={last24h} now={now} />
      <StatusDistribution deliveries={deliveries} />
    </>
  )}
</CardContent>
```

Recent-deliveries card — add an error branch ahead of the loading branch:

```tsx
{deliveriesError ? (
  <ErrorState title="Couldn't load deliveries" error={deliveriesErr} onRetry={() => refetchDeliveries()} />
) : deliveriesLoading ? (
  <RowSkeletons />
) : deliveries.length === 0 ? (
  /* unchanged empty state */
) : (
  /* unchanged list */
)}
```

Sources card — same shape:

```tsx
{sourcesError ? (
  <ErrorState title="Couldn't load sources" error={sourcesErr} onRetry={() => refetchSources()} />
) : sourcesLoading ? (
  <RowSkeletons />
) : sources.length === 0 ? (
  /* unchanged empty state */
) : (
  /* unchanged list */
)}
```

Add the import: `import { ErrorState } from "@/components/error-state"`.

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd web/ui && npx vitest run src/routes/dashboard.test.tsx`
Expected: PASS (all pre-existing dashboard tests plus the new one)

- [ ] **Step 5: Commit**

```bash
git add web/ui/src/routes/dashboard.tsx web/ui/src/routes/dashboard.test.tsx
git commit -m "feat(ui): dashboard shows inline errors and dashes instead of fake zeros"
```

---

### Task 6: DeliveryDetail — distinguish 404 from other failures

**Files:**
- Modify: `web/ui/src/routes/delivery-detail.tsx`
- Test: `web/ui/src/routes/delivery-detail.test.tsx` (append)

**Interfaces:**
- Consumes: `ErrorState` (Task 2), `ApiError` from `@/lib/api`.
- Behavior: 404 → "Delivery not found" (unchanged copy); any other query error → `ErrorState` with retry; attempts-query error → inline `ErrorState` above the body instead of silently showing zero attempts.

- [ ] **Step 1: Write the failing tests**

Append to `web/ui/src/routes/delivery-detail.test.tsx` (reuse its existing render/fetch helpers):

```tsx
test("shows error panel with retry for non-404 failures", async () => {
  vi.stubGlobal("fetch", vi.fn(() =>
    Promise.resolve(new Response("internal error", { status: 500 })),
  ))
  renderRoutes([{ path: "/deliveries/:id", element: <DeliveryDetail /> }], "/deliveries/d1")

  expect(await screen.findByText("Couldn't load delivery")).toBeInTheDocument()
  expect(screen.getByRole("button", { name: "Retry" })).toBeInTheDocument()
  expect(screen.queryByText("Delivery not found")).not.toBeInTheDocument()
})

test("still shows 'Delivery not found' for a 404", async () => {
  vi.stubGlobal("fetch", vi.fn(() =>
    Promise.resolve(new Response("delivery not found", { status: 404 })),
  ))
  renderRoutes([{ path: "/deliveries/:id", element: <DeliveryDetail /> }], "/deliveries/nope")

  expect(await screen.findByText("Delivery not found")).toBeInTheDocument()
  expect(screen.queryByRole("button", { name: "Retry" })).not.toBeInTheDocument()
})
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd web/ui && npx vitest run src/routes/delivery-detail.test.tsx`
Expected: FAIL — the 500 case currently renders "Delivery not found".

- [ ] **Step 3: Implement**

In `web/ui/src/routes/delivery-detail.tsx`:

```tsx
import { ApiError } from "@/lib/api"
import { ErrorState } from "@/components/error-state"
// ...
export function DeliveryDetail() {
  const { id = "" } = useParams()
  const { data: delivery, isLoading, isError, error, refetch } = useDelivery(id)
  const {
    data: attempts = [], isError: attemptsError,
    error: attemptsErr, refetch: refetchAttempts,
  } = useAttempts(id)
  const forwardDelivery = useForwardDelivery()

  if (isLoading) {
    return <div className="p-4 text-sm text-muted-foreground">Loading…</div>
  }

  const notFound = error instanceof ApiError && error.status === 404
  if (notFound || (!delivery && !isError)) {
    return <div className="p-4 text-sm text-muted-foreground">Delivery not found</div>
  }
  if (isError || !delivery) {
    return <ErrorState title="Couldn't load delivery" error={error} onRetry={() => refetch()} />
  }
  // ... rest unchanged, except the body:
```

Body — render an attempts error above `DeliveryDetailBody` when that query failed:

```tsx
{attemptsError && (
  <ErrorState title="Couldn't load attempts" error={attemptsErr} onRetry={() => refetchAttempts()} />
)}
<DeliveryDetailBody delivery={delivery} attempts={attempts} />
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd web/ui && npx vitest run src/routes/delivery-detail.test.tsx`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add web/ui/src/routes/delivery-detail.tsx web/ui/src/routes/delivery-detail.test.tsx
git commit -m "feat(ui): delivery detail distinguishes 404 from fetch failures"
```

---

### Task 7: Source detail + tabs (`SourceDetail`, `SourceActions`, `SourceEvents`, `SourceScript`) + changelog

**Files:**
- Modify: `web/ui/src/routes/source-detail.tsx`
- Modify: `web/ui/src/routes/source-actions.tsx`
- Modify: `web/ui/src/routes/source-events.tsx`
- Modify: `web/ui/src/routes/source-script.tsx`
- Modify: `CHANGELOG.md`
- Test: `web/ui/src/routes/source-actions.test.tsx` (create)
- Test: `web/ui/src/routes/source-detail.test.tsx` (create)

**Interfaces:**
- Consumes: `ErrorState` (Task 2), `ApiError` from `@/lib/api`.
- Note: `SourceOverview` and `SourceScript` share `useSource(slug)`'s query key with `SourceDetail`, so the parent's error handling covers the source fetch; only their *additional* queries need handling (`SourceScript`'s deliveries list).

- [ ] **Step 1: Write the failing tests**

Create `web/ui/src/routes/source-detail.test.tsx`:

```tsx
import { afterEach, expect, test, vi } from "vitest"
import { screen } from "@testing-library/react"
import { renderRoutes } from "@/test/render"
import { SourceDetail } from "./source-detail"

afterEach(() => vi.restoreAllMocks())

test("shows error panel for non-404 source fetch failures", async () => {
  vi.stubGlobal("fetch", vi.fn(() =>
    Promise.resolve(new Response("internal error", { status: 500 })),
  ))
  renderRoutes([{ path: "/sources/:slug", element: <SourceDetail /> }], "/sources/github")

  expect(await screen.findByText("Couldn't load source")).toBeInTheDocument()
  expect(screen.queryByText("Source not found")).not.toBeInTheDocument()
})

test("still shows 'Source not found' for a 404", async () => {
  vi.stubGlobal("fetch", vi.fn(() =>
    Promise.resolve(new Response("source not found", { status: 404 })),
  ))
  renderRoutes([{ path: "/sources/:slug", element: <SourceDetail /> }], "/sources/nope")

  expect(await screen.findByText("Source not found")).toBeInTheDocument()
})
```

Create `web/ui/src/routes/source-actions.test.tsx`:

```tsx
import { afterEach, expect, test, vi } from "vitest"
import { screen } from "@testing-library/react"
import { renderRoutes } from "@/test/render"
import { SourceActions } from "./source-actions"

afterEach(() => vi.restoreAllMocks())

test("shows error panel instead of 'No actions yet' when fetch fails", async () => {
  vi.stubGlobal("fetch", vi.fn(() =>
    Promise.resolve(new Response("internal error", { status: 500 })),
  ))
  renderRoutes([{ path: "/sources/:slug/actions", element: <SourceActions /> }], "/sources/github/actions")

  expect(await screen.findByText("Couldn't load actions")).toBeInTheDocument()
  expect(screen.queryByText(/No actions yet/)).not.toBeInTheDocument()
})
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd web/ui && npx vitest run src/routes/source-detail.test.tsx src/routes/source-actions.test.tsx`
Expected: FAIL — both currently render not-found/empty copy on failure.

- [ ] **Step 3: Implement**

`web/ui/src/routes/source-detail.tsx` — mirror the DeliveryDetail pattern:

```tsx
import { ApiError } from "@/lib/api"
import { ErrorState } from "@/components/error-state"
// ...
const { data: source, isLoading, isError, error, refetch } = useSource(slug)

if (isLoading) {
  return <div className="p-4 text-sm text-muted-foreground">Loading…</div>
}
const notFound = error instanceof ApiError && error.status === 404
if (notFound || (!source && !isError)) {
  return <div className="p-4 text-sm text-muted-foreground">Source not found</div>
}
if (isError || !source) {
  return <ErrorState title="Couldn't load source" error={error} onRetry={() => refetch()} />
}
```

`web/ui/src/routes/source-actions.tsx` — add an error branch to the existing ternary chain:

```tsx
import { ErrorState } from "@/components/error-state"
// ...
const { data: actions = [], isLoading, isError, error, refetch } = useActions(slug)
// ...
{isLoading ? (
  <div className="p-4 text-sm text-muted-foreground">Loading…</div>
) : isError ? (
  <ErrorState title="Couldn't load actions" error={error} onRetry={() => refetch()} />
) : actions.length === 0 ? (
  /* unchanged */
```

`web/ui/src/routes/source-events.tsx` — same shape:

```tsx
import { ErrorState } from "@/components/error-state"
// ...
const { data: deliveries = [], isLoading, isError, error, refetch } = useDeliveries(
  { source: slug },
  { refetchInterval: 3000 },
)
// ...
{isLoading ? (
  <div className="p-4 text-sm text-muted-foreground">Loading…</div>
) : isError ? (
  <ErrorState title="Couldn't load deliveries" error={error} onRetry={() => refetch()} />
) : deliveries.length === 0 ? (
  /* unchanged */
```

`web/ui/src/routes/source-script.tsx` — the deliveries list feeds the test-run picker; on error, say so instead of implying there are no deliveries:

```tsx
const { data: deliveries, isError: deliveriesError } = useDeliveries({ source: slug })
// ...
{deliveriesError ? (
  <p className="text-sm text-destructive">
    Couldn't load deliveries for test runs. Reload the page to try again.
  </p>
) : !hasDeliveries ? (
  /* unchanged "You need at least one recorded delivery…" */
```

`CHANGELOG.md` — under `## [Unreleased]` → `### Fixed` (create the group if absent):

```markdown
- Web UI now shows inline error panels with a Retry button when data fails to load, instead of misleading empty states ("No sources yet", "Delivery not found") when the API is unreachable; 4xx responses are no longer retried before surfacing.
```

- [ ] **Step 4: Run the full frontend suite**

Run: `cd web/ui && npx vitest run`
Expected: PASS — all suites, including pre-existing `source-script.test.tsx` and `sources-layout.test.tsx`.

- [ ] **Step 5: Commit**

```bash
git add web/ui/src/routes/source-detail.tsx web/ui/src/routes/source-actions.tsx web/ui/src/routes/source-events.tsx web/ui/src/routes/source-script.tsx web/ui/src/routes/source-detail.test.tsx web/ui/src/routes/source-actions.test.tsx CHANGELOG.md
git commit -m "feat(ui): inline query error handling for source detail and tabs"
```
