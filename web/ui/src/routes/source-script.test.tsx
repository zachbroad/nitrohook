import { afterEach, expect, test, vi } from "vitest"
import { waitFor } from "@testing-library/react"
import { renderRoutes } from "@/test/render"
import { DEFAULT_TRANSFORM_SCRIPT, seedScriptBody, SourceScript } from "./source-script"

function mockSource(script_body: string | null) {
  vi.stubGlobal("fetch", vi.fn().mockImplementation((url: string) => {
    if (url.includes("/api/deliveries")) {
      return Promise.resolve(new Response("[]", {
        status: 200, headers: { "Content-Type": "application/json" },
      }))
    }
    return Promise.resolve(new Response(JSON.stringify({
      id: "1", name: "GitHub", slug: "github", mode: "active", script_body,
      created_at: "2024-01-01T00:00:00Z", updated_at: "2024-01-02T00:00:00Z",
    }), { status: 200, headers: { "Content-Type": "application/json" } }))
  }))
}

afterEach(() => vi.restoreAllMocks())

test("seedScriptBody falls back to the scaffold for empty or missing scripts", () => {
  expect(seedScriptBody(null)).toBe(DEFAULT_TRANSFORM_SCRIPT)
  expect(seedScriptBody(undefined)).toBe(DEFAULT_TRANSFORM_SCRIPT)
  expect(seedScriptBody("")).toBe(DEFAULT_TRANSFORM_SCRIPT)
  expect(seedScriptBody("return event;")).toBe("return event;")
})

test("seeds the editor with the default scaffold when the source has no script", async () => {
  mockSource(null)
  const { container } = renderRoutes(
    [{ path: "/sources/:slug/script", element: <SourceScript /> }],
    "/sources/github/script",
  )
  await waitFor(() =>
    expect(container.querySelector(".cm-content")?.textContent).toContain("event.payload.processed = true;"),
  )
})

test("seeds the editor with the saved script when the source has one", async () => {
  mockSource("event.payload.custom = 1;\nreturn event;")
  const { container } = renderRoutes(
    [{ path: "/sources/:slug/script", element: <SourceScript /> }],
    "/sources/github/script",
  )
  await waitFor(() =>
    expect(container.querySelector(".cm-content")?.textContent).toContain("event.payload.custom = 1;"),
  )
})
