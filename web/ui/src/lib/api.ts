import type {
  Source, Action, Delivery, DeliveryAttempt, ScriptTestResult, ActionType, AuthPreset,
} from "./types"

const BASE = import.meta.env.VITE_API_BASE_URL ?? "http://localhost:8080"

export class ApiError extends Error {
  status: number
  constructor(status: number, message: string) { super(message); this.status = status; this.name = "ApiError" }
}

export async function apiFetch<T>(path: string, opts: RequestInit = {}): Promise<T> {
  const res = await fetch(BASE + path, {
    ...opts,
    headers: { "Content-Type": "application/json", ...(opts.headers ?? {}) },
  })
  if (!res.ok) throw new ApiError(res.status, (await res.text()) || res.statusText)
  if (res.status === 204) return undefined as T
  const ct = res.headers.get("Content-Type") ?? ""
  return (ct.includes("application/json") ? await res.json() : (await res.text())) as T
}

const j = (body: unknown): RequestInit => ({ body: JSON.stringify(body) })

export const listSources = () => apiFetch<Source[]>("/api/sources")
export const getSource = (slug: string) => apiFetch<Source>(`/api/sources/${slug}`)
export const createSource = (b: { name: string; mode?: string; script_body?: string }) =>
  apiFetch<Source>("/api/sources", { method: "POST", ...j(b) })
export const updateSource = (slug: string, b: Partial<Pick<Source, "mode" | "script_body">>) =>
  apiFetch<Source>(`/api/sources/${slug}`, { method: "PATCH", ...j(b) })
export const deleteSource = (slug: string) =>
  apiFetch<void>(`/api/sources/${slug}`, { method: "DELETE" })

export const listAuthPresets = () => apiFetch<AuthPreset[]>("/api/auth/presets")
export const updateSourceAuth = (
  slug: string,
  b: { enabled: boolean; preset?: string; secret?: string; public_key?: string },
) => apiFetch<Source>(`/api/sources/${slug}/auth`, { method: "PUT", ...j(b) })

export const listActions = (slug: string) => apiFetch<Action[]>(`/api/sources/${slug}/actions`)
export const createAction = (slug: string, b: Record<string, unknown> & { type: ActionType }) =>
  apiFetch<Action>(`/api/sources/${slug}/actions`, { method: "POST", ...j(b) })
export const updateAction = (slug: string, id: string, b: Record<string, unknown>) =>
  apiFetch<Action>(`/api/sources/${slug}/actions/${id}`, { method: "PATCH", ...j(b) })
export const deleteAction = (slug: string, id: string) =>
  apiFetch<void>(`/api/sources/${slug}/actions/${id}`, { method: "DELETE" })

export const testScript = (slug: string, b: { script_body: string; delivery_id: string }) =>
  apiFetch<ScriptTestResult>(`/api/sources/${slug}/script/test`, { method: "POST", ...j(b) })

export const listDeliveries = (p: { source?: string; limit?: number } = {}) => {
  const q = new URLSearchParams()
  if (p.source) q.set("source", p.source)
  if (p.limit) q.set("limit", String(p.limit))
  const qs = q.toString()
  return apiFetch<Delivery[]>(`/api/deliveries${qs ? `?${qs}` : ""}`)
}
export const getDelivery = (id: string) => apiFetch<Delivery>(`/api/deliveries/${id}`)
export const listAttempts = (id: string) => apiFetch<DeliveryAttempt[]>(`/api/deliveries/${id}/attempts`)
export const forwardDelivery = (id: string) =>
  apiFetch<{ status: string }>(`/api/deliveries/${id}/forward`, { method: "POST" })
export const forwardAll = (slug: string) =>
  apiFetch<{ forwarded: number }>(`/api/sources/${slug}/deliveries/forward-all`, { method: "POST" })

// Query retry policy: 4xx responses are deterministic (retrying a 404 three
// times just delays the error UI), so only network failures and 5xx retry.
export function shouldRetryQuery(failureCount: number, error: unknown): boolean {
  if (error instanceof ApiError && error.status < 500) return false
  return failureCount < 2
}
