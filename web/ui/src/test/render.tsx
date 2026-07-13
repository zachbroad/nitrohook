import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { render } from "@testing-library/react"
import { RouterProvider, createMemoryRouter, Outlet } from "react-router-dom"
import type { RouteObject } from "react-router-dom"
import { NuqsAdapter } from "nuqs/adapters/react-router/v7"

export function renderRoutes(routes: RouteObject[], initial = "/") {
  // nuqs's react-router adapter reads window.location.search, which a memory
  // router doesn't update — keep the jsdom URL in sync with the initial entry.
  window.history.replaceState(null, "", initial)
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  const router = createMemoryRouter(
    [{ element: <NuqsAdapter><Outlet /></NuqsAdapter>, children: routes }],
    { initialEntries: [initial] },
  )
  return render(<QueryClientProvider client={qc}><RouterProvider router={router} /></QueryClientProvider>)
}
