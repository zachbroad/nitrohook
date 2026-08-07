import { describe, expect, it, vi, beforeEach, test } from "vitest"
import { apiFetch, ApiError, shouldRetryQuery } from "./api"

beforeEach(() => { vi.restoreAllMocks() })

describe("apiFetch", () => {
  it("returns parsed JSON on 200", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(
      new Response(JSON.stringify({ ok: true }), { status: 200, headers: { "Content-Type": "application/json" } })
    ))
    await expect(apiFetch<{ ok: boolean }>("/api/x")).resolves.toEqual({ ok: true })
  })

  it("throws ApiError with server text on non-2xx", async () => {
    vi.stubGlobal("fetch", vi.fn().mockImplementation(() =>
      Promise.resolve(new Response("boom", { status: 400 }))
    ))
    await expect(apiFetch("/api/x")).rejects.toMatchObject({ status: 400, message: "boom" })
    await expect(apiFetch("/api/x")).rejects.toBeInstanceOf(ApiError)
  })
})

test("shouldRetryQuery skips 4xx errors", () => {
  expect(shouldRetryQuery(0, new ApiError(404, "not found"))).toBe(false)
  expect(shouldRetryQuery(0, new ApiError(400, "bad request"))).toBe(false)
})

test("shouldRetryQuery retries 5xx and network errors up to 2 times", () => {
  expect(shouldRetryQuery(0, new ApiError(500, "boom"))).toBe(true)
  expect(shouldRetryQuery(1, new TypeError("Failed to fetch"))).toBe(true)
  expect(shouldRetryQuery(2, new ApiError(500, "boom"))).toBe(false)
})
