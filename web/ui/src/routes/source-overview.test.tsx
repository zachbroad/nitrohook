import { afterEach, beforeEach, expect, test, vi } from "vitest"
import { screen } from "@testing-library/react"
import { Navigate } from "react-router-dom"
import { renderRoutes } from "@/test/render"
import { SourceDetail } from "./source-detail"
import { SourceOverview } from "./source-overview"

const json = (body: unknown) =>
  new Response(JSON.stringify(body), {
    status: 200, headers: { "Content-Type": "application/json" },
  })

beforeEach(() => {
  vi.stubGlobal("fetch", vi.fn((url: RequestInfo | URL) => {
    if (String(url).includes("/api/auth/presets"))
      return Promise.resolve(json([
        { name: "github", label: "GitHub", needs_secret: true, needs_public_key: false },
      ]))
    return Promise.resolve(json({
      id: "1", name: "GitHub", slug: "github", mode: "active",
      created_at: "2024-01-01T00:00:00Z", updated_at: "2024-01-02T00:00:00Z",
    }))
  }))
})
afterEach(() => vi.restoreAllMocks())

test("renders the ingest URL and mode for the source", async () => {
  renderRoutes([
    {
      path: "/sources/:slug",
      element: <SourceDetail />,
      children: [
        { index: true, element: <Navigate to="overview" replace /> },
        { path: "overview", element: <SourceOverview /> },
      ],
    },
  ], "/sources/github/overview")

  expect(await screen.findByText(/webhooks\/github/)).toBeInTheDocument()
  expect(screen.getByText(/active/i)).toBeInTheDocument()
})
