import { expect, test } from "vitest"
import { render } from "@testing-library/react"
import { JsonViewer } from "./json-viewer"

test("pretty-prints the data as formatted JSON", () => {
  const { container } = render(<JsonViewer data={{ a: 1 }} />)
  expect(container.textContent).toContain('"a": 1')
})

test("handles undefined data gracefully", () => {
  const { container } = render(<JsonViewer data={undefined} />)
  expect(container.textContent).toContain("undefined")
})

test("handles null data gracefully", () => {
  const { container } = render(<JsonViewer data={null} />)
  expect(container.textContent).toContain("null")
})
