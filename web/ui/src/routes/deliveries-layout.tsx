import * as React from "react"

import { AppShell } from "@/components/app-shell"
import { ListPane } from "@/components/list-pane"
import { StatusBadge } from "@/components/status-badge"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { useDeliveries } from "@/lib/queries"
import type { DeliveryStatus } from "@/lib/types"

const STATUS_OPTIONS: { value: DeliveryStatus | "all"; label: string }[] = [
  { value: "all", label: "All" },
  { value: "pending", label: "pending" },
  { value: "processing", label: "processing" },
  { value: "completed", label: "completed" },
  { value: "failed", label: "failed" },
  { value: "recorded", label: "recorded" },
]

export function DeliveriesLayout() {
  const { data: deliveries = [], isLoading } = useDeliveries({})
  const [status, setStatus] = React.useState<DeliveryStatus | "all">("all")

  const filtered =
    status === "all" ? deliveries : deliveries.filter((d) => d.status === status)

  return (
    <AppShell
      list={
        <ListPane
          header={
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
          }
          empty={isLoading ? "Loading…" : "No deliveries yet"}
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
