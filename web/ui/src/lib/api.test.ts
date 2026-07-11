import { describe, expect, it, vi, beforeEach } from "vitest"
import { apiFetch, ApiError } from "./api"

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
