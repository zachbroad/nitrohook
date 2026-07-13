import { useParams, useSearchParams } from "react-router-dom"

import { Button } from "@/components/ui/button"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { StatusBadge } from "@/components/status-badge"
import { DeliveryDetailBody } from "@/components/delivery-detail-body"
import {
  useAttempts,
  useDelivery,
  useDeliveries,
  useForwardAll,
  useForwardDelivery,
} from "@/lib/queries"
import type { Delivery } from "@/lib/types"
import { formatDate } from "@/lib/utils"

function DeliveryModal({
  id,
  onOpenChange,
}: {
  id: string
  onOpenChange: (open: boolean) => void
}) {
  const { data: delivery, isLoading } = useDelivery(id)
  const { data: attempts = [] } = useAttempts(id)

  return (
    <Dialog open onOpenChange={onOpenChange}>
      <DialogContent className="max-h-[85vh] max-w-2xl overflow-y-auto sm:max-w-2xl">
        {isLoading || !delivery ? (
          <div className="p-4 text-sm text-muted-foreground">Loading…</div>
        ) : (
          <>
            <DialogHeader>
              <div className="flex items-center gap-2">
                <DialogTitle>Delivery {delivery.id.slice(0, 8)}</DialogTitle>
                <StatusBadge status={delivery.status} />
              </div>
              <p className="font-mono text-xs text-muted-foreground">
                {delivery.idempotency_key}
              </p>
            </DialogHeader>
            <DeliveryDetailBody delivery={delivery} attempts={attempts} />
          </>
        )}
      </DialogContent>
    </Dialog>
  )
}

export function SourceEvents() {
  const { slug = "" } = useParams()
  const [searchParams, setSearchParams] = useSearchParams()
  const { data: deliveries = [], isLoading } = useDeliveries(
    { source: slug },
    { refetchInterval: 3000 },
  )
  const forwardDelivery = useForwardDelivery()
  const forwardAll = useForwardAll(slug)

  const hasRecorded = deliveries.some((d: Delivery) => d.status === "recorded")
  const selectedId = searchParams.get("delivery")

  const openDelivery = (id: string) => {
    setSearchParams((prev) => {
      const next = new URLSearchParams(prev)
      next.set("delivery", id)
      return next
    })
  }
  const closeDelivery = () => {
    setSearchParams((prev) => {
      const next = new URLSearchParams(prev)
      next.delete("delivery")
      return next
    })
  }

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
              <TableHead className="text-right">Retries</TableHead>
              <TableHead className="text-right">Actions</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {deliveries.map((delivery: Delivery) => (
              <TableRow
                key={delivery.id}
                className="cursor-pointer hover:bg-muted/50"
                onClick={() => openDelivery(delivery.id)}
              >
                <TableCell className="text-sm text-muted-foreground">
                  {formatDate(delivery.received_at)}
                </TableCell>
                <TableCell>
                  <StatusBadge status={delivery.status} />
                </TableCell>
                <TableCell className="max-w-xs truncate font-mono text-xs text-muted-foreground">
                  {delivery.idempotency_key}
                </TableCell>
                <TableCell className="text-right text-sm text-muted-foreground">
                  {delivery.retry_count}
                </TableCell>
                <TableCell className="text-right">
                  {delivery.status === "recorded" && (
                    <Button
                      type="button"
                      variant="outline"
                      size="sm"
                      disabled={forwardDelivery.isPending}
                      onClick={(e) => {
                        e.stopPropagation()
                        forwardDelivery.mutate(delivery.id)
                      }}
                    >
                      Forward
                    </Button>
                  )}
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      )}

      {selectedId && (
        <DeliveryModal
          id={selectedId}
          onOpenChange={(open) => {
            if (!open) closeDelivery()
          }}
        />
      )}
    </div>
  )
}
