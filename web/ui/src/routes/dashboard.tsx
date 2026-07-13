import { Link } from "react-router-dom"
import { ArrowRightIcon, PlusIcon } from "lucide-react"

import { Header } from "@/components/header"
import { StatusBadge } from "@/components/status-badge"
import { Badge } from "@/components/ui/badge"
import { buttonVariants } from "@/components/ui/button"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Skeleton } from "@/components/ui/skeleton"
import { useDeliveries, useSources } from "@/lib/queries"
import type { Delivery, DeliveryStatus, Source } from "@/lib/types"
import { cn, timeAgo } from "@/lib/utils"

// Stats window: the API has no aggregate endpoint, so everything on this page
// is derived from the most recent deliveries only.
const WINDOW_LIMIT = 200
const HOUR_MS = 3_600_000

const STATUS_ORDER: DeliveryStatus[] = ["completed", "failed", "processing", "pending", "recorded"]
const STATUS_DOT: Record<DeliveryStatus, string> = {
  completed: "bg-emerald-500",
  failed: "bg-destructive",
  processing: "bg-amber-300",
  pending: "bg-amber-500",
  recorded: "bg-slate-400",
}

export function Dashboard() {
  const { data: sources = [], isLoading: sourcesLoading } = useSources()
  const { data: deliveries = [], isLoading: deliveriesLoading } = useDeliveries(
    { limit: WINDOW_LIMIT },
    { refetchInterval: 10_000 },
  )
  const loading = sourcesLoading || deliveriesLoading

  const now = Date.now()
  const last24h = deliveries.filter((d) => now - new Date(d.received_at).getTime() < 24 * HOUR_MS)
  const failed = deliveries.filter((d) => d.status === "failed").length
  const completed = deliveries.filter((d) => d.status === "completed").length
  const terminal = completed + failed
  const successRate = terminal === 0 ? null : Math.round((completed / terminal) * 100)
  const activeSources = sources.filter((s) => s.mode === "active").length

  const sourceNames = new Map(sources.map((s) => [s.id, s.name]))
  const deliveriesBySource = new Map<string, number>()
  for (const d of deliveries)
    deliveriesBySource.set(d.source_id, (deliveriesBySource.get(d.source_id) ?? 0) + 1)

  return (
    <div className="flex h-screen w-screen flex-col overflow-hidden">
      <Header />
      <main className="flex-1 overflow-y-auto">
        <div className="mx-auto flex w-full max-w-5xl flex-col gap-6 px-6 py-8">
          <div>
            <h1 className="text-xl font-semibold tracking-tight">Overview</h1>
            <p className="text-sm text-muted-foreground">
              Based on the {WINDOW_LIMIT} most recent deliveries.
            </p>
          </div>

          <div className="grid grid-cols-2 gap-4 lg:grid-cols-4">
            <StatTile
              label="Sources"
              value={loading ? null : String(sources.length)}
              hint={`${activeSources} active · ${sources.length - activeSources} recording`}
            />
            <StatTile
              label="Deliveries · 24h"
              value={loading ? null : String(last24h.length)}
              hint="received in the last day"
            />
            <StatTile
              label="Failed"
              value={loading ? null : String(failed)}
              hint={failed > 0 ? "needs attention" : "all clear"}
              alert={failed > 0}
              to="/deliveries?status=failed"
            />
            <StatTile
              label="Success rate"
              value={loading ? null : successRate === null ? "—" : `${successRate}%`}
              hint={terminal === 0 ? "no completed deliveries yet" : `of ${terminal} finished deliveries`}
            />
          </div>

          <Card>
            <CardHeader className="flex-row items-baseline justify-between">
              <CardTitle className="text-sm font-medium">Activity · last 24 hours</CardTitle>
              <span className="text-xs text-muted-foreground">{last24h.length} deliveries</span>
            </CardHeader>
            <CardContent className="flex flex-col gap-4">
              <HourlyBars deliveries={last24h} now={now} />
              <StatusDistribution deliveries={deliveries} />
            </CardContent>
          </Card>

          <div className="grid gap-6 lg:grid-cols-2">
            <Card>
              <CardHeader className="flex-row items-baseline justify-between">
                <CardTitle className="text-sm font-medium">Recent deliveries</CardTitle>
                <ViewAll to="/deliveries" />
              </CardHeader>
              <CardContent className="flex flex-col gap-1">
                {deliveriesLoading ? (
                  <RowSkeletons />
                ) : deliveries.length === 0 ? (
                  <p className="py-4 text-sm text-muted-foreground">
                    No deliveries yet. Send a webhook to a source's ingest URL to see it here.
                  </p>
                ) : (
                  deliveries.slice(0, 8).map((d) => (
                    <Link
                      key={d.id}
                      to={`/deliveries/${d.id}`}
                      className="flex items-center gap-3 rounded-md px-2 py-1.5 transition-colors hover:bg-accent/50"
                    >
                      <span className="font-mono text-xs">{d.id.slice(0, 8)}</span>
                      <span className="min-w-0 flex-1 truncate text-sm text-muted-foreground">
                        {sourceNames.get(d.source_id) ?? "unknown source"}
                      </span>
                      <StatusBadge status={d.status} />
                      <span className="w-16 shrink-0 text-right text-xs text-muted-foreground">
                        {timeAgo(d.received_at, now)}
                      </span>
                    </Link>
                  ))
                )}
              </CardContent>
            </Card>

            <Card>
              <CardHeader className="flex-row items-baseline justify-between">
                <CardTitle className="text-sm font-medium">Sources</CardTitle>
                <ViewAll to="/sources" />
              </CardHeader>
              <CardContent className="flex flex-col gap-1">
                {sourcesLoading ? (
                  <RowSkeletons />
                ) : sources.length === 0 ? (
                  <div className="flex flex-col items-start gap-3 py-4">
                    <p className="text-sm text-muted-foreground">
                      Create a source to start receiving webhooks.
                    </p>
                    <Link to="/sources" className={buttonVariants({ size: "sm" })}>
                      <PlusIcon className="size-4" /> Create a source
                    </Link>
                  </div>
                ) : (
                  sources.map((s) => <SourceRow key={s.id} source={s} count={deliveriesBySource.get(s.id) ?? 0} />)
                )}
              </CardContent>
            </Card>
          </div>
        </div>
      </main>
    </div>
  )
}

function StatTile({ label, value, hint, alert = false, to }: {
  label: string; value: string | null; hint: string; alert?: boolean; to?: string
}) {
  const tile = (
    <Card className={cn("gap-1 py-4", to && "transition-colors hover:bg-accent/50")}>
      <CardContent className="flex flex-col gap-1 px-4">
        <span className="text-xs font-medium uppercase tracking-wide text-muted-foreground">{label}</span>
        {value === null ? (
          <Skeleton className="h-8 w-16" />
        ) : (
          <span className={cn("text-2xl font-semibold tabular-nums", alert && "text-destructive")}>
            {value}
          </span>
        )}
        <span className="text-xs text-muted-foreground">{hint}</span>
      </CardContent>
    </Card>
  )
  return to ? <Link to={to} className="rounded-xl">{tile}</Link> : tile
}

function HourlyBars({ deliveries, now }: { deliveries: Delivery[]; now: number }) {
  // Bucket 0 is 24h ago, bucket 23 is the current hour.
  const buckets = Array.from({ length: 24 }, () => 0)
  for (const d of deliveries) {
    const age = now - new Date(d.received_at).getTime()
    const i = 23 - Math.min(23, Math.floor(age / HOUR_MS))
    buckets[i]++
  }
  const max = Math.max(1, ...buckets)

  return (
    <div className="flex flex-col gap-1">
      <div className="flex h-24 items-end gap-0.5" role="img" aria-label="Deliveries per hour, last 24 hours">
        {buckets.map((count, i) => {
          const hour = new Date(now - (23 - i) * HOUR_MS).getHours()
          return (
            <div key={i} className="group relative flex h-full flex-1 flex-col justify-end">
              <div className="pointer-events-none absolute bottom-full left-1/2 z-10 mb-1 hidden -translate-x-1/2 whitespace-nowrap rounded-md border bg-popover px-2 py-1 text-xs text-popover-foreground shadow-sm group-hover:block">
                {String(hour).padStart(2, "0")}:00 — {count} {count === 1 ? "delivery" : "deliveries"}
              </div>
              <div
                className={cn(
                  "rounded-t-[3px] transition-colors",
                  count > 0 ? "bg-primary/70 group-hover:bg-primary" : "bg-muted",
                )}
                style={{ height: count > 0 ? `${Math.max(6, (count / max) * 100)}%` : "2px" }}
              />
            </div>
          )
        })}
      </div>
      <div className="flex justify-between text-xs text-muted-foreground">
        <span>24h ago</span>
        <span>now</span>
      </div>
    </div>
  )
}

function StatusDistribution({ deliveries }: { deliveries: Delivery[] }) {
  const counts = new Map<DeliveryStatus, number>()
  for (const d of deliveries) counts.set(d.status, (counts.get(d.status) ?? 0) + 1)
  const present = STATUS_ORDER.filter((s) => (counts.get(s) ?? 0) > 0)
  if (present.length === 0) return null

  return (
    <div className="flex flex-col gap-2">
      <div className="flex h-2 gap-0.5 overflow-hidden rounded-full">
        {present.map((s) => (
          <div
            key={s}
            className={STATUS_DOT[s]}
            style={{ width: `${((counts.get(s) ?? 0) / deliveries.length) * 100}%` }}
          />
        ))}
      </div>
      <div className="flex flex-wrap gap-x-4 gap-y-1">
        {present.map((s) => (
          <Link
            key={s}
            to={`/deliveries?status=${s}`}
            className="flex items-center gap-1.5 text-xs text-muted-foreground transition-colors hover:text-foreground"
          >
            <span className={cn("size-2 rounded-full", STATUS_DOT[s])} />
            {s} <span className="tabular-nums text-foreground">{counts.get(s)}</span>
          </Link>
        ))}
      </div>
    </div>
  )
}

function SourceRow({ source, count }: { source: Source; count: number }) {
  return (
    <Link
      to={`/sources/${source.slug}`}
      className="flex items-center gap-3 rounded-md px-2 py-1.5 transition-colors hover:bg-accent/50"
    >
      <span className="min-w-0 flex-1 truncate text-sm">{source.name}</span>
      <Badge variant="secondary">{source.mode}</Badge>
      <span className="w-24 shrink-0 text-right text-xs tabular-nums text-muted-foreground">
        {count} {count === 1 ? "delivery" : "deliveries"}
      </span>
    </Link>
  )
}

function ViewAll({ to }: { to: string }) {
  return (
    <Link to={to} className="flex items-center gap-1 text-xs text-muted-foreground transition-colors hover:text-foreground">
      View all <ArrowRightIcon className="size-3" />
    </Link>
  )
}

function RowSkeletons() {
  return (
    <div className="flex flex-col gap-2 py-1">
      {Array.from({ length: 4 }, (_, i) => <Skeleton key={i} className="h-7 w-full" />)}
    </div>
  )
}
