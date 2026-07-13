import { expect, test } from "vitest"
import { ApiError } from "./api"
import { describeApiError } from "./utils"

test("describeApiError returns the ApiError message", () => {
  expect(describeApiError(new ApiError(500, "internal error"))).toBe("internal error")
})

test("describeApiError maps network TypeError to a reachability message", () => {
  expect(describeApiError(new TypeError("Failed to fetch"))).toBe(
    "Can't reach the API. Check that the server is running.",
  )
})

test("describeApiError falls back for unknown values", () => {
  expect(describeApiError(undefined)).toBe("An unexpected error occurred.")
  expect(describeApiError("boom")).toBe("An unexpected error occurred.")
})
