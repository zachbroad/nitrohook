import { afterEach, beforeEach, expect, test, vi } from "vitest"
import { render, screen } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { ActionForm } from "./action-form"

function renderForm() {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return render(
    <QueryClientProvider client={qc}>
      <ActionForm slug="github" />
    </QueryClientProvider>
  )
}

beforeEach(() => {
  vi.stubGlobal(
    "fetch",
    vi.fn().mockResolvedValue(
      new Response("[]", { status: 200, headers: { "Content-Type": "application/json" } })
    )
  )
})
afterEach(() => vi.restoreAllMocks())

test("the type select drives which fields render (create mode)", async () => {
  const user = userEvent.setup()
  renderForm()

  await user.click(screen.getByRole("button", { name: /new action/i }))

  // Default type is webhook, so its field should be present.
  expect(await screen.findByLabelText(/target url/i)).toBeInTheDocument()

  await user.click(screen.getByRole("combobox", { name: /type/i }))
  await user.click(await screen.findByRole("option", { name: /slack/i }))

  expect(await screen.findByLabelText(/webhook url/i)).toBeInTheDocument()
  expect(screen.queryByLabelText(/target url/i)).not.toBeInTheDocument()

  await user.click(screen.getByRole("combobox", { name: /type/i }))
  await user.click(await screen.findByRole("option", { name: /^webhook$/i }))

  expect(await screen.findByLabelText(/target url/i)).toBeInTheDocument()
  expect(screen.queryByLabelText(/webhook url/i)).not.toBeInTheDocument()
})
