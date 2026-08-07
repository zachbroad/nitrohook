import { afterEach, beforeEach, expect, test, vi } from "vitest"
import { screen } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { renderRoutes } from "@/test/render"
import { SourcesLayout } from "./sources-layout"
import { EmptyState } from "./empty-state"

let fetchMock: ReturnType<typeof vi.fn>

beforeEach(() => {
  fetchMock = vi.fn().mockImplementation((_url: string, opts?: RequestInit) => {
    if (opts?.method === "POST") {
      return Promise.resolve(
        new Response(
          JSON.stringify({
            id: "2",
            name: "Stripe",
            slug: "stripe",
            mode: "active",
            created_at: "",
            updated_at: "",
          }),
          { status: 201, headers: { "Content-Type": "application/json" } }
        )
      )
    }
    return Promise.resolve(
      new Response(JSON.stringify([]), {
        status: 200,
        headers: { "Content-Type": "application/json" },
      })
    )
  })
  vi.stubGlobal("fetch", fetchMock)
})
afterEach(() => vi.restoreAllMocks())

test("creating a source submits the form and navigates to the new source", async () => {
  const user = userEvent.setup()
  renderRoutes(
    [
      {
        path: "/sources",
        element: <SourcesLayout />,
        children: [{ index: true, element: <EmptyState title="Select a source" /> }],
      },
      { path: "/sources/:slug", element: <div>source detail</div> },
    ],
    "/sources"
  )

  await user.click(await screen.findByRole("button", { name: /new source/i }))
  await user.type(await screen.findByLabelText("Name"), "Stripe")
  await user.click(screen.getByRole("button", { name: /create/i }))

  expect(await screen.findByText("source detail")).toBeInTheDocument()

  const postCall = fetchMock.mock.calls.find(([, opts]) => opts?.method === "POST")
  expect(postCall).toBeTruthy()
  expect(JSON.parse((postCall![1] as RequestInit).body as string)).toEqual({
    name: "Stripe",
    mode: "active",
  })
})
