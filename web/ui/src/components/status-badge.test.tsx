import { expect, test } from "vitest"
import { render } from "@testing-library/react"
import { StatusBadge } from "./status-badge"

test("renders completed status with success styling", () => {
  const { getByText } = render(<StatusBadge status="completed" />)
  const el = getByText("completed")
  expect(el.className).toContain("bg-emerald-500/15")
})

test("renders success status with success styling", () => {
  const { getByText } = render(<StatusBadge status="success" />)
  const el = getByText("success")
  expect(el.className).toContain("bg-emerald-500/15")
})

test("renders failed status with destructive styling", () => {
  const { getByText } = render(<StatusBadge status="failed" />)
  const el = getByText("failed")
  expect(el.className).toContain("bg-destructive")
})

test("renders pending status with warning styling", () => {
  const { getByText } = render(<StatusBadge status="pending" />)
  const el = getByText("pending")
  expect(el.className).toContain("bg-amber-500/15")
})

test("renders processing status with warning styling", () => {
  const { getByText } = render(<StatusBadge status="processing" />)
  const el = getByText("processing")
  expect(el.className).toContain("bg-amber-500/15")
})

test("renders recorded status with muted styling", () => {
  const { getByText } = render(<StatusBadge status="recorded" />)
  const el = getByText("recorded")
  expect(el.className).toContain("bg-slate-500/15")
})
