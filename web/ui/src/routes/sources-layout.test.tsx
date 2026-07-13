import { afterEach, beforeEach, expect, test, vi } from "vitest"
import { screen } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
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

test("shows error panel instead of 'No sources yet' when fetch fails, and retries", async () => {
  const fetchMock = vi.fn((): Promise<Response> => Promise.reject(new TypeError("Failed to fetch")))
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
