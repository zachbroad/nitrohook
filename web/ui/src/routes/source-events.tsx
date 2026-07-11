import { Link, useParams } from "react-router-dom"

import { Button } from "@/components/ui/button"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import { StatusBadge } from "@/components/status-badge"
import { useDeliveries, useForwardAll, useForwardDelivery } from "@/lib/queries"
import type { Delivery } from "@/lib/types"
import { formatDate } from "@/lib/utils"

export function SourceEvents() {
  const { slug = "" } = useParams()
  const { data: deliveries = [], isLoading } = useDeliveries({ source: slug })
  const forwardDelivery = useForwardDelivery()
  const forwardAll = useForwardAll(slug)

  const hasRecorded = deliveries.some((d: Delivery) => d.status === "recorded")

  return (
    <div className="flex flex-col gap-4">
      <div className="flex items-center justify-between">
        <h2 className="text-sm font-medium text-muted-foreground">Events</h2>
        <Button
          type="button"
          variant="outline"
          size="sm"
          disabled={!hasRecorded || forwardAll.isPending}
          onClick={() => forwardAll.mutate()}
        >
          Forward all
        </Button>
      </div>

      {isLoading ? (
        <div className="p-4 text-sm text-muted-foreground">Loading…</div>
      ) : deliveries.length === 0 ? (
        <div className="p-4 text-sm text-muted-foreground">
          No deliveries recorded yet.
        </div>
      ) : (
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Received</TableHead>
              <TableHead>Status</TableHead>
              <TableHead>Idempotency key</TableHead>
              <TableHead className="text-right">Actions</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {deliveries.map((delivery: Delivery) => (
              <TableRow key={delivery.id}>
                <TableCell className="text-sm text-muted-foreground">
                  {formatDate(delivery.received_at)}
                </TableCell>
                <TableCell>
                  <StatusBadge status={delivery.status} />
                </TableCell>
                <TableCell className="max-w-xs truncate font-mono text-xs text-muted-foreground">
                  {delivery.idempotency_key}
                </TableCell>
                <TableCell className="text-right">
                  <div className="flex justify-end gap-2">
                    {delivery.status === "recorded" && (
                      <Button
                        type="button"
                        variant="outline"
                        size="sm"
                        disabled={forwardDelivery.isPending}
                        onClick={() => forwardDelivery.mutate(delivery.id)}
                      >
                        Forward
                      </Button>
                    )}
                    <Button type="button" variant="ghost" size="sm" render={
                      <Link to={`/deliveries/${delivery.id}`} />
                    }>
                      View
                    </Button>
                  </div>
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      )}
    </div>
  )
}
