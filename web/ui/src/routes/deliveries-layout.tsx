import { parseAsStringLiteral, useQueryState } from "nuqs"

import { AppShell } from "@/components/app-shell"
import { ErrorState } from "@/components/error-state"
import { ListPane } from "@/components/list-pane"
import { StatusBadge } from "@/components/status-badge"
import { Input } from "@/components/ui/input"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { useDeliveries } from "@/lib/queries"
import type { DeliveryStatus } from "@/lib/types"

const STATUS_FILTER = ["all", "pending", "processing", "completed", "failed", "recorded"] as const

const STATUS_OPTIONS: { value: DeliveryStatus | "all"; label: string }[] = [
  { value: "all", label: "All" },
  { value: "pending", label: "pending" },
  { value: "processing", label: "processing" },
  { value: "completed", label: "completed" },
  { value: "failed", label: "failed" },
  { value: "recorded", label: "recorded" },
]

export function DeliveriesLayout() {
  const { data: deliveries = [], isLoading, isError, error, refetch } = useDeliveries({})
  const [status, setStatus] = useQueryState(
    "status",
    parseAsStringLiteral(STATUS_FILTER).withDefault("all"),
  )
  const [search, setSearch] = useQueryState("q", { defaultValue: "" })

  const query = search.trim().toLowerCase()
  const filtered = deliveries
    .filter((d) => status === "all" || d.status === status)
    .filter((d) => !query || d.id.toLowerCase().includes(query) || d.idempotency_key.toLowerCase().includes(query))

  return (
    <AppShell
      list={
        <ListPane
          header={
            <div className="flex flex-col gap-2">
              <Select
                value={status}
                onValueChange={(v) => setStatus(v as DeliveryStatus | "all")}
              >
                <SelectTrigger className="w-full">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {STATUS_OPTIONS.map((o) => (
                    <SelectItem key={o.value} value={o.value}>
                      {o.label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
              <Input
                placeholder="Search deliveries…"
                value={search}
                onChange={(e) => setSearch(e.target.value || null)}
              />
            </div>
          }
          error={isError
            ? <ErrorState title="Couldn't load deliveries" error={error} onRetry={() => refetch()} />
            : undefined}
          empty={isLoading ? "Loading…" : query ? "No matching deliveries" : "No deliveries yet"}
          items={filtered.map((d) => ({
            key: d.id,
            to: `/deliveries/${d.id}`,
            label: (
              <span className="flex items-center justify-between gap-2">
                <span className="truncate font-mono text-xs">{d.id.slice(0, 8)}</span>
                <StatusBadge status={d.status} />
              </span>
            ),
          }))}
        />
      }
    />
  )
}
