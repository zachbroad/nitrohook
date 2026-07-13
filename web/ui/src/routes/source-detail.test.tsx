import { afterEach, expect, test, vi } from "vitest"
import { screen } from "@testing-library/react"
import { renderRoutes } from "@/test/render"
import { SourceDetail } from "./source-detail"

afterEach(() => vi.restoreAllMocks())

test("shows error panel for non-404 source fetch failures", async () => {
  vi.stubGlobal("fetch", vi.fn(() =>
    Promise.resolve(new Response("internal error", { status: 500 })),
  ))
  renderRoutes([{ path: "/sources/:slug", element: <SourceDetail /> }], "/sources/github")

  expect(await screen.findByText("Couldn't load source")).toBeInTheDocument()
  expect(screen.queryByText("Source not found")).not.toBeInTheDocument()
})

test("still shows 'Source not found' for a 404", async () => {
  vi.stubGlobal("fetch", vi.fn(() =>
    Promise.resolve(new Response("source not found", { status: 404 })),
  ))
  renderRoutes([{ path: "/sources/:slug", element: <SourceDetail /> }], "/sources/nope")

  expect(await screen.findByText("Source not found")).toBeInTheDocument()
})
