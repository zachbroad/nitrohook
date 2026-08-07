import { useParams } from "react-router-dom"

import { Button } from "@/components/ui/button"
import { StatusBadge } from "@/components/status-badge"
import { DeliveryDetailBody } from "@/components/delivery-detail-body"
import { ErrorState } from "@/components/error-state"
import { ApiError } from "@/lib/api"
import { useAttempts, useDelivery, useForwardDelivery } from "@/lib/queries"
import { formatDate } from "@/lib/utils"

export function DeliveryDetail() {
  const { id = "" } = useParams()
  const { data: delivery, isLoading, isError, error, refetch } = useDelivery(id)
  const {
    data: attempts = [], isError: attemptsError,
    error: attemptsErr, refetch: refetchAttempts,
  } = useAttempts(id)
  const forwardDelivery = useForwardDelivery()

  if (isLoading) {
    return <div className="p-4 text-sm text-muted-foreground">Loading…</div>
  }

  const notFound = error instanceof ApiError && error.status === 404
  if (notFound || (!delivery && !isError)) {
    return <div className="p-4 text-sm text-muted-foreground">Delivery not found</div>
  }
  if (isError || !delivery) {
    return <ErrorState title="Couldn't load delivery" error={error} onRetry={() => refetch()} />
  }

  return (
    <div className="flex flex-col gap-6 p-6">
      <div className="flex items-start justify-between gap-4">
        <div className="flex flex-col gap-1">
          <div className="flex items-center gap-2">
            <h1 className="text-lg font-semibold">Delivery {delivery.id.slice(0, 8)}</h1>
            <StatusBadge status={delivery.status} />
          </div>
          <p className="text-sm text-muted-foreground">Source: {delivery.source_id}</p>
          <p className="text-sm text-muted-foreground">Received: {formatDate(delivery.received_at)}</p>
          <p className="font-mono text-xs text-muted-foreground">{delivery.idempotency_key}</p>
        </div>
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
      </div>

      {attemptsError && (
        <ErrorState title="Couldn't load attempts" error={attemptsErr} onRetry={() => refetchAttempts()} />
      )}
      <DeliveryDetailBody delivery={delivery} attempts={attempts} />
    </div>
  )
}
