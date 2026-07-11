import { afterEach, beforeEach, expect, test, vi } from "vitest"
import { screen } from "@testing-library/react"
import { renderRoutes } from "@/test/render"
import { DeliveryDetail } from "./delivery-detail"

beforeEach(() => {
  vi.stubGlobal(
    "fetch",
    vi.fn().mockImplementation((url: string) => {
      if (url.includes("/attempts")) {
        return Promise.resolve(
          new Response(
            JSON.stringify([
              {
                id: "att-1",
                delivery_id: "abc",
                action_id: "act-1",
                attempt_number: 1,
                status: "success",
                response_status: 200,
                created_at: "2024-01-01T00:00:00Z",
              },
            ]),
            { status: 200, headers: { "Content-Type": "application/json" } }
          )
        )
      }
      return Promise.resolve(
        new Response(
          JSON.stringify({
            id: "abc",
            source_id: "src-1",
            idempotency_key: "key-1",
            headers: { "x-foo": "bar" },
            payload: { hello: "world" },
            status: "completed",
            received_at: "2024-01-01T00:00:00Z",
          }),
          { status: 200, headers: { "Content-Type": "application/json" } }
        )
      )
    })
  )
})
afterEach(() => vi.restoreAllMocks())

test("renders delivery payload and attempts", async () => {
  renderRoutes(
    [{ path: "/deliveries/:id", element: <DeliveryDetail /> }],
    "/deliveries/abc"
  )

  expect(await screen.findByText(/"hello"/)).toBeInTheDocument()
  expect(await screen.findByText(/attempt 1/i)).toBeInTheDocument()
})
