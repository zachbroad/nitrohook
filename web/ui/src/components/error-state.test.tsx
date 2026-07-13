import { expect, test, vi } from "vitest"
import { render, screen } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { ApiError } from "@/lib/api"
import { ErrorState } from "./error-state"

test("renders title, error detail, and calls onRetry", async () => {
  const onRetry = vi.fn()
  render(<ErrorState title="Couldn't load sources" error={new ApiError(500, "internal error")} onRetry={onRetry} />)
  expect(screen.getByText("Couldn't load sources")).toBeInTheDocument()
  expect(screen.getByText("internal error")).toBeInTheDocument()
  await userEvent.click(screen.getByRole("button", { name: "Retry" }))
  expect(onRetry).toHaveBeenCalledOnce()
})

test("omits the retry button when onRetry is not given", () => {
  render(<ErrorState title="Couldn't load sources" error={new TypeError("Failed to fetch")} />)
  expect(screen.queryByRole("button", { name: "Retry" })).not.toBeInTheDocument()
})
