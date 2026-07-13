import { StatusBadge } from "@/components/status-badge"
import { JsonViewer } from "@/components/json-viewer"
import type { Delivery, DeliveryAttempt } from "@/lib/types"
import { formatDate } from "@/lib/utils"

export function DeliveryDetailBody({
  delivery,
  attempts,
}: {
  delivery: Delivery
  attempts: DeliveryAttempt[]
}) {
  return (
    <div className="flex flex-col gap-6">
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
            {attempts.map((a) => (
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
