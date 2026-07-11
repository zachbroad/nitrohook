import { render, screen } from "@testing-library/react"
import { expect, test } from "vitest"

test("harness renders", () => {
  render(<h1>NitroHook</h1>)
  expect(screen.getByText("NitroHook")).toBeInTheDocument()
})
