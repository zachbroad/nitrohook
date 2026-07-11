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
