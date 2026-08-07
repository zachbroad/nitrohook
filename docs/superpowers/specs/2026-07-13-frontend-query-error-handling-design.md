# Frontend Query Error Handling — Design

**Date:** 2026-07-13
**Status:** Approved

## Problem

In the React SPA (`web/ui`), mutations already surface failures via toasts and
route rendering errors hit `RouteError` boundaries, but **query (read) failures
are invisible**. Every consumer destructures with a default
(`const { data: sources = [] } = useSources()`), so when the API is down or
returns 5xx the UI renders misleading empty states ("No sources yet",
"No deliveries yet", "Delivery not found") instead of an error.

## Decision

Surface query failures **inline, in place of the data, with a Retry button**.
Page chrome stays usable; this matches the existing skeleton/empty-state
patterns. (Rejected: throwing to the route boundary — a single failed panel
would blow away the page; toast-only — empty data would still look legitimate.
Also rejected: a generic `<QueryBoundary>` render-prop wrapper — more
abstraction than eight call sites justify.)

## Components

### 1. `ErrorState` component — `src/components/error-state.tsx`

Visual sibling of `EmptyState`: centered muted panel with a heading
("Couldn't load sources"), a human-readable detail line, and a small
**Retry** button wired to React Query's `refetch`.

Props: `{ title: string; error: unknown; onRetry?: () => void }`.

### 2. `describeApiError(err)` helper — `src/lib/utils.ts`

Maps errors to display text:
- `ApiError` → its message (and status available for callers).
- Network failure (`TypeError` from fetch) → "Can't reach the API".
- Anything else → "An unexpected error occurred."

### 3. Per-view `isError` handling

| View | Behavior on query error |
|------|------------------------|
| `Dashboard` | Affected card content replaced by `ErrorState`; stat tiles show "—" instead of fake zeros when their backing query errored |
| `SourcesLayout` / `DeliveriesLayout` | `ListPane` gains an optional `error` slot rendered instead of the empty message, with retry |
| `DeliveryDetail` | `ApiError.status === 404` → keep "Delivery not found"; any other error → `ErrorState` + retry (today all failures show "not found") |
| Source tabs (`source-overview`, `source-actions`, `source-events`, `source-script`) | `isError` → `ErrorState` + retry for source/actions/deliveries queries |

### 4. QueryClient retry defaults — `src/main.tsx`

```ts
new QueryClient({
  defaultOptions: {
    queries: {
      retry: (failureCount, err) =>
        !(err instanceof ApiError && err.status < 500) && failureCount < 2,
    },
  },
})
```

Don't retry 4xx (deterministic failures like 404); retry network/5xx up to 2
times. Today the default retries 404s three times before settling.

## Testing

Extend existing Vitest suites (`dashboard.test.tsx`, `sources-layout.test.tsx`,
`delivery-detail.test.tsx`, plus new coverage where a view has none) with a
failing-fetch case asserting:
- the error panel renders (and the misleading empty state does not),
- clicking Retry refetches and renders data on success,
- `DeliveryDetail` still shows "Delivery not found" for a 404.

Test-render helper (`src/test/render.tsx`) must disable query retries so error
tests don't wait on retry backoff.

## Out of scope

- Toasting background refetch failures.
- Offline detection / global connectivity banner.
- Any backend changes.
