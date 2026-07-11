import { useParams } from "react-router-dom"
import { toast } from "sonner"
import { Button } from "@/components/ui/button"
import { Label } from "@/components/ui/label"
import { Switch } from "@/components/ui/switch"
import { useSource, useUpdateSource } from "@/lib/queries"

const API_BASE = import.meta.env.VITE_API_BASE_URL ?? "http://localhost:8080"

function formatDate(iso: string) {
  const d = new Date(iso)
  return Number.isNaN(d.getTime()) ? iso : d.toLocaleString()
}

export function SourceOverview() {
  const { slug = "" } = useParams()
  const { data: source } = useSource(slug)
  const updateSource = useUpdateSource(slug)

  if (!source) return null

  const ingestUrl = `${API_BASE}/webhooks/${slug}`
  const isActive = source.mode === "active"

  const copyUrl = async () => {
    try {
      await navigator.clipboard.writeText(ingestUrl)
      toast.success("Copied to clipboard")
    } catch {
      toast.error("Could not copy URL")
    }
  }

  return (
    <div className="flex max-w-xl flex-col gap-6">
      <div className="grid gap-1.5">
        <Label htmlFor="source-mode-switch">Mode</Label>
        <div className="flex items-center gap-3">
          <Switch
            id="source-mode-switch"
            checked={isActive}
            onCheckedChange={(checked) =>
              updateSource.mutate({ mode: checked ? "active" : "record" })
            }
          />
          <span className="text-sm">
            {isActive
              ? "Active — fans out to actions immediately"
              : "Record — stores deliveries only, no fan-out"}
          </span>
        </div>
      </div>

      <div className="grid gap-1.5">
        <Label>Webhook ingest URL</Label>
        <div className="flex items-center gap-2">
          <code className="flex-1 truncate rounded border bg-muted px-2 py-1.5 text-sm">
            {ingestUrl}
          </code>
          <Button type="button" variant="outline" size="sm" onClick={copyUrl}>
            Copy
          </Button>
        </div>
      </div>

      <dl className="grid grid-cols-[auto_1fr] gap-x-4 gap-y-1 text-sm">
        <dt className="text-muted-foreground">Created</dt>
        <dd>{formatDate(source.created_at)}</dd>
        <dt className="text-muted-foreground">Updated</dt>
        <dd>{formatDate(source.updated_at)}</dd>
      </dl>
    </div>
  )
}
