import { useMutation, useQuery, useQueryClient, type UseQueryOptions } from "@tanstack/react-query"
import { toast } from "sonner"
import * as api from "./api"
import { ApiError } from "./api"

export const qk = {
  sources: ["sources"] as const,
  source: (s: string) => ["source", s] as const,
  actions: (s: string) => ["source", s, "actions"] as const,
  deliveries: (p: { source?: string; limit?: number } = {}) => ["deliveries", p] as const,
  delivery: (id: string) => ["delivery", id] as const,
  attempts: (id: string) => ["delivery", id, "attempts"] as const,
}

const onError = (e: unknown) =>
  toast.error(e instanceof ApiError ? e.message : "Request failed")

export const useSources = () => useQuery({ queryKey: qk.sources, queryFn: api.listSources })
export const useSource = (slug: string) =>
  useQuery({ queryKey: qk.source(slug), queryFn: () => api.getSource(slug), enabled: !!slug })
export const useActions = (slug: string) =>
  useQuery({ queryKey: qk.actions(slug), queryFn: () => api.listActions(slug), enabled: !!slug })
export const useDeliveries = (
  p: { source?: string; limit?: number } = {},
  opts: Partial<UseQueryOptions<Awaited<ReturnType<typeof api.listDeliveries>>>> = {},
) => useQuery({ queryKey: qk.deliveries(p), queryFn: () => api.listDeliveries(p), ...opts })
export const useDelivery = (id: string) =>
  useQuery({ queryKey: qk.delivery(id), queryFn: () => api.getDelivery(id), enabled: !!id })
export const useAttempts = (id: string) =>
  useQuery({ queryKey: qk.attempts(id), queryFn: () => api.listAttempts(id), enabled: !!id })

export function useCreateSource() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: api.createSource, onError,
    onSuccess: () => {
      toast.success("Source created")
      qc.invalidateQueries({ queryKey: qk.sources })
    },
  })
}
export function useUpdateSource(slug: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (b: Parameters<typeof api.updateSource>[1]) => api.updateSource(slug, b), onError,
    onSuccess: () => {
      toast.success("Changes saved")
      qc.invalidateQueries({ queryKey: qk.source(slug) })
      qc.invalidateQueries({ queryKey: qk.sources })
    },
  })
}
export function useDeleteSource() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: api.deleteSource, onError,
    onSuccess: () => {
      toast.success("Source deleted")
      qc.invalidateQueries({ queryKey: qk.sources })
    },
  })
}
export function useCreateAction(slug: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (b: Parameters<typeof api.createAction>[1]) => api.createAction(slug, b), onError,
    onSuccess: () => {
      toast.success("Action created")
      qc.invalidateQueries({ queryKey: qk.actions(slug) })
    },
  })
}
export function useUpdateAction(slug: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (v: { id: string; body: Record<string, unknown> }) => api.updateAction(slug, v.id, v.body), onError,
    onSuccess: () => {
      toast.success("Action updated")
      qc.invalidateQueries({ queryKey: qk.actions(slug) })
    },
  })
}
export function useDeleteAction(slug: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => api.deleteAction(slug, id), onError,
    onSuccess: () => {
      toast.success("Action deleted")
      qc.invalidateQueries({ queryKey: qk.actions(slug) })
    },
  })
}
export function useForwardDelivery() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: api.forwardDelivery,
    onError,
    onSuccess: (_data, id) => {
      toast.success("Delivery forwarded")
      // Forwarding creates a new attempt and changes the delivery's status,
      // so refresh the list plus this delivery's own detail and attempts.
      qc.invalidateQueries({ queryKey: ["deliveries"] })
      qc.invalidateQueries({ queryKey: qk.delivery(id) })
      qc.invalidateQueries({ queryKey: qk.attempts(id) })
    },
  })
}
export function useForwardSelected() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: async (ids: string[]) => {
      const results = await Promise.allSettled(ids.map((id) => api.forwardDelivery(id)))
      const failedIds = ids.filter((_, i) => results[i].status === "rejected")
      const firstError = results.find(
        (r): r is PromiseRejectedResult => r.status === "rejected",
      )?.reason
      return { total: ids.length, failedIds, firstError }
    },
    onSuccess: ({ total, failedIds, firstError }) => {
      const forwarded = total - failedIds.length
      if (forwarded > 0)
        toast.success(`Forwarded ${forwarded} ${forwarded === 1 ? "delivery" : "deliveries"}`)
      if (failedIds.length > 0) {
        const message = firstError instanceof ApiError ? firstError.message : "Request failed"
        toast.error(`Failed to forward ${failedIds.length} of ${total}: ${message}`)
      }
      qc.invalidateQueries({ queryKey: ["deliveries"] })
      qc.invalidateQueries({ queryKey: ["delivery"] })
    },
  })
}

export function useForwardAll(slug: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: () => api.forwardAll(slug),
    onError,
    onSuccess: (r) => {
      toast.success(`Forwarded ${r.forwarded} deliveries`)
      qc.invalidateQueries({ queryKey: ["deliveries"] })
    },
  })
}
