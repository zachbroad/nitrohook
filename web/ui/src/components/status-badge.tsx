import { Badge } from "@/components/ui/badge"
import type { AttemptStatus, DeliveryStatus } from "@/lib/types"

export type BadgeableStatus = DeliveryStatus | AttemptStatus

export const STATUS_CLASSNAMES: Record<BadgeableStatus, string> = {
  completed: "bg-emerald-500/15 text-emerald-700 dark:text-emerald-400",
  success: "bg-emerald-500/15 text-emerald-700 dark:text-emerald-400",
  failed: "bg-destructive/10 text-destructive dark:bg-destructive/20",
  pending: "bg-amber-500/15 text-amber-700 dark:text-amber-400",
  processing: "bg-amber-500/15 text-amber-700 dark:text-amber-400",
  recorded: "bg-slate-500/15 text-slate-700 dark:text-slate-400",
}

export function StatusBadge({ status }: { status: BadgeableStatus }) {
  return (
    <Badge variant="secondary" className={STATUS_CLASSNAMES[status]}>
      {status}
    </Badge>
  )
}
