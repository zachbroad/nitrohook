import { afterEach, expect, test, vi } from "vitest"
import { screen } from "@testing-library/react"
import { renderRoutes } from "@/test/render"
import { SourceActions } from "./source-actions"

afterEach(() => vi.restoreAllMocks())

test("shows error panel instead of 'No actions yet' when fetch fails", async () => {
  vi.stubGlobal("fetch", vi.fn(() =>
    Promise.resolve(new Response("internal error", { status: 500 })),
  ))
  renderRoutes([{ path: "/sources/:slug/actions", element: <SourceActions /> }], "/sources/github/actions")

  expect(await screen.findByText("Couldn't load actions")).toBeInTheDocument()
  expect(screen.queryByText(/No actions yet/)).not.toBeInTheDocument()
})
