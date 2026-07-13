import { afterEach, beforeEach, expect, test, vi } from "vitest"
import { screen } from "@testing-library/react"
import { renderRoutes } from "@/test/render"
import { Dashboard } from "./dashboard"

const sources = [
  { id: "s1", name: "GitHub", slug: "github", mode: "active", created_at: "", updated_at: "" },
  { id: "s2", name: "Stripe", slug: "stripe", mode: "record", created_at: "", updated_at: "" },
]
const deliveries = [
  { id: "d1-abcdef00", source_id: "s1", idempotency_key: "k1", headers: {}, payload: {},
    status: "completed", received_at: new Date().toISOString(), retry_count: 0 },
  { id: "d2-abcdef00", source_id: "s1", idempotency_key: "k2", headers: {}, payload: {},
    status: "failed", received_at: new Date().toISOString(), retry_count: 2 },
  { id: "d3-abcdef00", source_id: "s2", idempotency_key: "k3", headers: {}, payload: {},
    status: "recorded", received_at: new Date().toISOString(), retry_count: 0 },
]

const json = (body: unknown) =>
  new Response(JSON.stringify(body), { status: 200, headers: { "Content-Type": "application/json" } })

beforeEach(() => {
  vi.stubGlobal("fetch", vi.fn((input: RequestInfo | URL) => {
    const url = String(input)
    if (url.includes("/api/deliveries")) return Promise.resolve(json(deliveries))
    if (url.includes("/api/sources")) return Promise.resolve(json(sources))
    return Promise.resolve(json([]))
  }))
})
afterEach(() => vi.restoreAllMocks())

test("renders stat tiles computed from sources and deliveries", async () => {
  renderRoutes([{ path: "/", element: <Dashboard /> }], "/")
  expect(await screen.findByText("1 active · 1 recording")).toBeInTheDocument()
  // 1 completed of 2 terminal deliveries → 50%
  expect(await screen.findByText("50%")).toBeInTheDocument()
})

test("lists recent deliveries with their source names", async () => {
  renderRoutes([{ path: "/", element: <Dashboard /> }], "/")
  expect(await screen.findByText("d1-abcde")).toBeInTheDocument()
  expect(screen.getAllByText("GitHub").length).toBeGreaterThan(0)
  expect(screen.getAllByText("failed").length).toBeGreaterThan(0)
})

test("failed tile and status legend deep-link to the filtered deliveries list", async () => {
  renderRoutes([{ path: "/", element: <Dashboard /> }], "/")
  await screen.findByText("50%")
  const failedLinks = screen
    .getAllByRole("link")
    .filter((a) => a.getAttribute("href") === "/deliveries?status=failed")
  // one from the Failed stat tile, one from the status-distribution legend
  expect(failedLinks.length).toBe(2)
  expect(
    screen.getAllByRole("link").some((a) => a.getAttribute("href") === "/deliveries?status=completed"),
  ).toBe(true)
})

test("shows create-source invitation when no sources exist", async () => {
  vi.stubGlobal("fetch", vi.fn(() => Promise.resolve(json([]))))
  renderRoutes([{ path: "/", element: <Dashboard /> }], "/")
  expect(await screen.findByText("Create a source")).toBeInTheDocument()
})
