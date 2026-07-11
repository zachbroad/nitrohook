import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { render } from "@testing-library/react"
import { RouterProvider, createMemoryRouter } from "react-router-dom"
import type { RouteObject } from "react-router-dom"

export function renderRoutes(routes: RouteObject[], initial = "/") {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  const router = createMemoryRouter(routes, { initialEntries: [initial] })
  return render(<QueryClientProvider client={qc}><RouterProvider router={router} /></QueryClientProvider>)
}
