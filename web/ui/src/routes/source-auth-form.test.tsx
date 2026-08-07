import { afterEach, beforeEach, expect, test, vi } from "vitest"
import { screen } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { renderRoutes } from "@/test/render"
import type { Source } from "@/lib/types"
import { SourceAuthForm } from "./source-auth-form"

const PRESETS = [
  { name: "github", label: "GitHub", needs_secret: true, needs_public_key: false },
  { name: "discord-ed25519", label: "Discord (Ed25519)", needs_secret: false, needs_public_key: true },
]

const baseSource: Source = {
  id: "1", name: "GitHub", slug: "github", mode: "active",
  created_at: "2024-01-01T00:00:00Z", updated_at: "2024-01-02T00:00:00Z",
}

const json = (body: unknown) =>
  new Response(JSON.stringify(body), {
    status: 200, headers: { "Content-Type": "application/json" },
  })

let fetchMock: ReturnType<typeof vi.fn>

beforeEach(() => {
  fetchMock = vi.fn((url: RequestInfo | URL) => {
    if (String(url).includes("/api/auth/presets")) return Promise.resolve(json(PRESETS))
    return Promise.resolve(json(baseSource))
  })
  vi.stubGlobal("fetch", fetchMock)
})
afterEach(() => vi.restoreAllMocks())

function renderForm(source: Source) {
  return renderRoutes([{ path: "/", element: <SourceAuthForm source={source} /> }], "/")
}

test("shows disabled state when the source has no auth config", async () => {
  renderForm(baseSource)
  expect(
    await screen.findByText(/accepted without verification/i),
  ).toBeInTheDocument()
  expect(screen.getByRole("switch")).not.toBeChecked()
  expect(screen.queryByLabelText(/shared secret/i)).not.toBeInTheDocument()
})

test("shows saved provider and secret hint when auth is configured", async () => {
  renderForm({
    ...baseSource,
    auth_config: {
      enabled: true, scheme: "hmac", preset: "github", has_secret: true,
    },
  })
  expect(await screen.findByText("GitHub")).toBeInTheDocument()
  expect(screen.getByRole("switch")).toBeChecked()
  expect(
    screen.getByPlaceholderText(/a secret is saved/i),
  ).toBeInTheDocument()
})

test("refuses to save an enabled config without a secret", async () => {
  const user = userEvent.setup()
  renderForm(baseSource)
  await user.click(await screen.findByRole("switch"))
  await user.click(screen.getByRole("button", { name: /save authentication/i }))
  expect(
    await screen.findByText(/a secret is required/i),
  ).toBeInTheDocument()
  expect(fetchMock).not.toHaveBeenCalledWith(
    expect.stringContaining("/auth"),
    expect.objectContaining({ method: "PUT" }),
  )
})

test("saving with a secret sends the config to the API", async () => {
  const user = userEvent.setup()
  renderForm(baseSource)
  await user.click(await screen.findByRole("switch"))
  await user.type(
    await screen.findByLabelText(/shared secret/i),
    "topsecret",
  )
  await user.click(screen.getByRole("button", { name: /save authentication/i }))

  const call = fetchMock.mock.calls.find(([url]) =>
    String(url).endsWith("/api/sources/github/auth"),
  )
  expect(call).toBeDefined()
  expect(call![1]).toMatchObject({ method: "PUT" })
  expect(JSON.parse(call![1].body)).toEqual({
    enabled: true, preset: "github", secret: "topsecret", public_key: "",
  })
})

test("disabling auth sends enabled:false", async () => {
  const user = userEvent.setup()
  renderForm({
    ...baseSource,
    auth_config: {
      enabled: true, scheme: "hmac", preset: "github", has_secret: true,
    },
  })
  await user.click(await screen.findByRole("switch"))
  await user.click(screen.getByRole("button", { name: /save authentication/i }))

  const call = fetchMock.mock.calls.find(([url]) =>
    String(url).endsWith("/api/sources/github/auth"),
  )
  expect(call).toBeDefined()
  expect(JSON.parse(call![1].body)).toEqual({ enabled: false })
})
