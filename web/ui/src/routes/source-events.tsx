import * as React from "react"
import { useParams, useSearchParams } from "react-router-dom"

import { Button } from "@/components/ui/button"
import { Checkbox } from "@/components/ui/checkbox"
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
import { ErrorState } from "@/components/error-state"
import {
  useAttempts,
  useDelivery,
  useDeliveries,
  useForwardAll,
  useForwardSelected,
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
  const { data: deliveries = [], isLoading, isError, error, refetch } = useDeliveries(
    { source: slug },
    { refetchInterval: 3000 },
  )
  const forwardSelected = useForwardSelected()
  const forwardAll = useForwardAll(slug)

  const [selected, setSelected] = React.useState<Set<string>>(new Set())
  // Only recorded deliveries can be forwarded; a delivery may leave that state
  // between refetches, so always derive the effective selection from the data.
  const recordedIds = deliveries
    .filter((d: Delivery) => d.status === "recorded")
    .map((d: Delivery) => d.id)
  const selectedRecorded = recordedIds.filter((id) => selected.has(id))
  const allSelected = recordedIds.length > 0 && selectedRecorded.length === recordedIds.length

  const toggleOne = (id: string, checked: boolean) => {
    setSelected((prev) => {
      const next = new Set(prev)
      if (checked) next.add(id)
      else next.delete(id)
      return next
    })
  }
  const toggleAll = (checked: boolean) => {
    setSelected(checked ? new Set(recordedIds) : new Set())
  }

  const hasRecorded = recordedIds.length > 0
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
        <div className="flex items-center gap-2">
          <Button
            type="button"
            variant="outline"
            size="sm"
            disabled={selectedRecorded.length === 0 || forwardSelected.isPending}
            onClick={() =>
              forwardSelected.mutate(selectedRecorded, {
                onSuccess: ({ failedIds }) => setSelected(new Set(failedIds)),
              })
            }
          >
            Forward selected{selectedRecorded.length > 0 && ` (${selectedRecorded.length})`}
          </Button>
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
      </div>

      {isLoading ? (
        <div className="p-4 text-sm text-muted-foreground">Loading…</div>
      ) : isError ? (
        <ErrorState title="Couldn't load deliveries" error={error} onRetry={() => refetch()} />
      ) : deliveries.length === 0 ? (
        <div className="p-4 text-sm text-muted-foreground">
          No deliveries recorded yet.
        </div>
      ) : (
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead className="w-8">
                <Checkbox
                  aria-label="Select all forwardable deliveries"
                  disabled={!hasRecorded}
                  checked={allSelected}
                  indeterminate={selectedRecorded.length > 0 && !allSelected}
                  onCheckedChange={(checked) => toggleAll(checked === true)}
                />
              </TableHead>
              <TableHead>Received</TableHead>
              <TableHead>Status</TableHead>
              <TableHead>Idempotency key</TableHead>
              <TableHead className="text-right">Retries</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {deliveries.map((delivery: Delivery) => (
              <TableRow
                key={delivery.id}
                className="cursor-pointer hover:bg-muted/50"
                onClick={() => openDelivery(delivery.id)}
              >
                <TableCell onClick={(e) => e.stopPropagation()}>
                  {delivery.status === "recorded" && (
                    <Checkbox
                      aria-label={`Select delivery ${delivery.id.slice(0, 8)}`}
                      checked={selected.has(delivery.id)}
                      onCheckedChange={(checked) => toggleOne(delivery.id, checked === true)}
                    />
                  )}
                </TableCell>
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
