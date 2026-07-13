import { afterEach, beforeEach, expect, test, vi } from "vitest"
import { screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { toast } from "sonner"
import { renderRoutes } from "@/test/render"
import { SourceEvents } from "./source-events"

vi.mock("sonner", () => ({ toast: { success: vi.fn(), error: vi.fn() } }))

const deliveries = [
  { id: "d1-recorded0", source_id: "s1", idempotency_key: "k1", headers: {}, payload: {},
    status: "recorded", received_at: new Date().toISOString(), retry_count: 0 },
  { id: "d2-recorded0", source_id: "s1", idempotency_key: "k2", headers: {}, payload: {},
    status: "recorded", received_at: new Date().toISOString(), retry_count: 0 },
  { id: "d3-completed", source_id: "s1", idempotency_key: "k3", headers: {}, payload: {},
    status: "completed", received_at: new Date().toISOString(), retry_count: 0 },
]

const json = (body: unknown) =>
  new Response(JSON.stringify(body), { status: 200, headers: { "Content-Type": "application/json" } })

let fetchMock: ReturnType<typeof vi.fn>

beforeEach(() => {
  fetchMock = vi.fn((input: RequestInfo | URL, opts?: RequestInit) => {
    const url = String(input)
    if (opts?.method === "POST" && url.includes("/forward")) return Promise.resolve(json({ status: "ok" }))
    if (url.includes("/api/deliveries")) return Promise.resolve(json(deliveries))
    return Promise.resolve(json([]))
  })
  vi.stubGlobal("fetch", fetchMock)
})
afterEach(() => vi.restoreAllMocks())

const routes = [{ path: "/sources/:slug/events", element: <SourceEvents /> }]

test("only recorded deliveries get a selection checkbox", async () => {
  renderRoutes(routes, "/sources/test/events")
  expect(await screen.findByLabelText("Select delivery d1-recor")).toBeInTheDocument()
  expect(screen.getByLabelText("Select delivery d2-recor")).toBeInTheDocument()
  expect(screen.queryByLabelText("Select delivery d3-compl")).not.toBeInTheDocument()
})

test("select-all + forward selected posts a forward per recorded delivery", async () => {
  const user = userEvent.setup()
  renderRoutes(routes, "/sources/test/events")

  await user.click(await screen.findByLabelText("Select all forwardable deliveries"))
  const button = screen.getByRole("button", { name: "Forward selected (2)" })
  await user.click(button)

  const forwardCalls = fetchMock.mock.calls
    .filter(([, opts]) => (opts as RequestInit | undefined)?.method === "POST")
    .map(([url]) => String(url))
  expect(forwardCalls.some((u) => u.endsWith("/api/deliveries/d1-recorded0/forward"))).toBe(true)
  expect(forwardCalls.some((u) => u.endsWith("/api/deliveries/d2-recorded0/forward"))).toBe(true)
  expect(forwardCalls).toHaveLength(2)
})

test("forward selected is disabled until something is selected", async () => {
  const user = userEvent.setup()
  renderRoutes(routes, "/sources/test/events")

  const button = await screen.findByRole("button", { name: "Forward selected" })
  expect(button).toBeDisabled()

  await user.click(await screen.findByLabelText("Select delivery d1-recor"))
  expect(screen.getByRole("button", { name: "Forward selected (1)" })).toBeEnabled()
})

test("partial forward-selected failure keeps only the failed delivery selected", async () => {
  fetchMock = vi.fn((input: RequestInfo | URL, opts?: RequestInit): Promise<Response> => {
    const url = String(input)
    if (opts?.method === "POST" && url.endsWith("/d1-recorded0/forward")) return Promise.resolve(json({ status: "ok" }))
    if (opts?.method === "POST" && url.endsWith("/d2-recorded0/forward"))
      return Promise.resolve(new Response("forward failed", { status: 500 }))
    if (url.includes("/api/deliveries")) return Promise.resolve(json(deliveries))
    return Promise.resolve(json([]))
  })
  vi.stubGlobal("fetch", fetchMock)

  const user = userEvent.setup()
  renderRoutes(routes, "/sources/test/events")

  await user.click(await screen.findByLabelText("Select all forwardable deliveries"))
  await user.click(screen.getByRole("button", { name: "Forward selected (2)" }))

  await waitFor(() =>
    expect(vi.mocked(toast.error)).toHaveBeenCalledWith("Failed to forward 1 of 2: forward failed"),
  )
  expect(screen.getByLabelText("Select delivery d1-recor")).not.toBeChecked()
  expect(screen.getByLabelText("Select delivery d2-recor")).toBeChecked()
})
