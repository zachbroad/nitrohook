import { afterEach, beforeEach, expect, test, vi } from "vitest"
import { screen } from "@testing-library/react"
import { renderRoutes } from "@/test/render"
import { DeliveriesLayout } from "./deliveries-layout"

const deliveries = [
  { id: "d1-abcdef00", source_id: "s1", idempotency_key: "k1", headers: {}, payload: {},
    status: "completed", received_at: new Date().toISOString(), retry_count: 0 },
  { id: "d2-abcdef00", source_id: "s1", idempotency_key: "k2", headers: {}, payload: {},
    status: "failed", received_at: new Date().toISOString(), retry_count: 2 },
]

const json = (body: unknown) =>
  new Response(JSON.stringify(body), { status: 200, headers: { "Content-Type": "application/json" } })

beforeEach(() => {
  vi.stubGlobal("fetch", vi.fn((input: RequestInfo | URL) => {
    const url = String(input)
    if (url.includes("/api/deliveries")) return Promise.resolve(json(deliveries))
    return Promise.resolve(json([]))
  }))
})
afterEach(() => vi.restoreAllMocks())

const routes = [{ path: "/deliveries", element: <DeliveriesLayout />, children: [{ index: true, element: <div /> }] }]

test("shows all deliveries without a status param", async () => {
  renderRoutes(routes, "/deliveries")
  expect(await screen.findByText("d1-abcde")).toBeInTheDocument()
  expect(screen.getByText("d2-abcde")).toBeInTheDocument()
})

test("?status= deep link filters the delivery list", async () => {
  renderRoutes(routes, "/deliveries?status=failed")
  expect(await screen.findByText("d2-abcde")).toBeInTheDocument()
  expect(screen.queryByText("d1-abcde")).not.toBeInTheDocument()
})

test("an invalid ?status= value falls back to showing everything", async () => {
  renderRoutes(routes, "/deliveries?status=bogus")
  expect(await screen.findByText("d1-abcde")).toBeInTheDocument()
  expect(screen.getByText("d2-abcde")).toBeInTheDocument()
})

test("shows error panel instead of 'No deliveries yet' when fetch fails", async () => {
  vi.stubGlobal("fetch", vi.fn(() =>
    Promise.resolve(new Response("internal error", { status: 500 })),
  ))
  renderRoutes(routes, "/deliveries")

  expect(await screen.findByText("Couldn't load deliveries")).toBeInTheDocument()
  expect(screen.getByText("internal error")).toBeInTheDocument()
  expect(screen.queryByText("No deliveries yet")).not.toBeInTheDocument()
})
