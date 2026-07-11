import { useParams } from "react-router-dom"

import { Button } from "@/components/ui/button"
import { StatusBadge } from "@/components/status-badge"
import { JsonViewer } from "@/components/json-viewer"
import { useAttempts, useDelivery, useForwardDelivery } from "@/lib/queries"
import type { DeliveryAttempt } from "@/lib/types"
import { formatDate } from "@/lib/utils"

export function DeliveryDetail() {
  const { id = "" } = useParams()
  const { data: delivery, isLoading } = useDelivery(id)
  const { data: attempts = [] } = useAttempts(id)
  const forwardDelivery = useForwardDelivery()

  if (isLoading) {
    return <div className="p-4 text-sm text-muted-foreground">Loading…</div>
  }

  if (!delivery) {
    return <div className="p-4 text-sm text-muted-foreground">Delivery not found</div>
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

      <div className="grid gap-4 sm:grid-cols-2">
        <div className="flex flex-col gap-1.5">
          <h2 className="text-sm font-medium text-muted-foreground">Headers</h2>
          <JsonViewer data={delivery.headers} />
        </div>
        <div className="flex flex-col gap-1.5">
          <h2 className="text-sm font-medium text-muted-foreground">Payload</h2>
          <JsonViewer data={delivery.payload} />
        </div>
        {delivery.transformed_headers != null && (
          <div className="flex flex-col gap-1.5">
            <h2 className="text-sm font-medium text-muted-foreground">Transformed headers</h2>
            <JsonViewer data={delivery.transformed_headers} />
          </div>
        )}
        {delivery.transformed_payload != null && (
          <div className="flex flex-col gap-1.5">
            <h2 className="text-sm font-medium text-muted-foreground">Transformed payload</h2>
            <JsonViewer data={delivery.transformed_payload} />
          </div>
        )}
      </div>

      <div className="flex flex-col gap-2">
        <h2 className="text-sm font-medium text-muted-foreground">Attempts</h2>
        {attempts.length === 0 ? (
          <p className="text-sm text-muted-foreground">No attempts yet.</p>
        ) : (
          <ul className="flex flex-col gap-2">
            {attempts.map((a: DeliveryAttempt) => (
              <li key={a.id} className="flex flex-col gap-1 rounded-lg border p-3 text-sm">
                <div className="flex items-center gap-2">
                  <span className="font-medium">Attempt {a.attempt_number}</span>
                  <StatusBadge status={a.status} />
                  {a.response_status != null && (
                    <span className="text-xs text-muted-foreground">HTTP {a.response_status}</span>
                  )}
                </div>
                {a.error_message && (
                  <p className="text-xs text-destructive">{a.error_message}</p>
                )}
                {a.next_retry_at && (
                  <p className="text-xs text-muted-foreground">
                    Next retry: {formatDate(a.next_retry_at)}
                  </p>
                )}
                <p className="text-xs text-muted-foreground">{formatDate(a.created_at)}</p>
              </li>
            ))}
          </ul>
        )}
      </div>
    </div>
  )
}
