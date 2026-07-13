import * as React from "react"
import { useParams } from "react-router-dom"
import { useMutation } from "@tanstack/react-query"
import { toast } from "sonner"

import { Button } from "@/components/ui/button"
import { Label } from "@/components/ui/label"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { ScriptEditor } from "@/components/script-editor"
import { JsonViewer } from "@/components/json-viewer"
import { useSource, useUpdateSource, useDeliveries } from "@/lib/queries"
import { testScript, ApiError } from "@/lib/api"

function formatDeliveryLabel(id: string, receivedAt: string) {
  const shortId = id.slice(0, 8)
  const d = new Date(receivedAt)
  const when = Number.isNaN(d.getTime()) ? receivedAt : d.toLocaleString()
  return `${shortId} — ${when}`
}

/**
 * Starter scaffold shown when a source has no saved transform script. Mirrors
 * the default the legacy HTML template rendered (web/templates/source-script.html)
 * so the editor is never blank on a fresh source.
 */
export const DEFAULT_TRANSFORM_SCRIPT = `function transform(event) {
  // event.payload  — the JSON body (object)
  // event.headers  — captured headers (object)
  // event.actions  — [{id, target_url}, ...]

  // Transform the payload
  event.payload.processed = true;

  // Return null to drop the event
  // Filter event.actions to route selectively

  return event;
}`

/**
 * Seeds the editor from a saved script, falling back to the scaffold when the
 * source has none. Matches the template's truthiness check: an empty string
 * (e.g. after Clear) reverts to the scaffold rather than showing a blank editor.
 */
export function seedScriptBody(scriptBody: string | null | undefined): string {
  return scriptBody ? scriptBody : DEFAULT_TRANSFORM_SCRIPT
}

export function SourceScript() {
  const { slug = "" } = useParams()
  const { data: source } = useSource(slug)
  const updateSource = useUpdateSource(slug)
  const { data: deliveries } = useDeliveries({ source: slug })

  const [scriptBody, setScriptBody] = React.useState("")
  const [deliveryId, setDeliveryId] = React.useState<string | null>(null)
  const seededSlug = React.useRef<string | null>(null)

  // Seed the editor from the loaded source once per slug. Keying the effect on
  // source.script_body instead would re-clobber in-progress edits every time a
  // save-triggered refetch resolves (Save PATCHes, invalidates, refetches, and
  // the stale-until-resolved value would overwrite newly typed characters).
  React.useEffect(() => {
    if (source && seededSlug.current !== slug) {
      setScriptBody(seedScriptBody(source.script_body))
      seededSlug.current = slug
    }
  }, [source, slug])

  const testRun = useMutation({
    mutationFn: () => testScript(slug, { script_body: scriptBody, delivery_id: deliveryId ?? "" }),
    onError: (e: unknown) => toast.error(e instanceof ApiError ? e.message : "Test run failed"),
  })

  const handleSave = () => updateSource.mutate({ script_body: scriptBody })
  const handleClear = () => {
    setScriptBody("")
    updateSource.mutate({ script_body: "" })
  }

  const hasDeliveries = (deliveries?.length ?? 0) > 0

  return (
    <div className="flex max-w-3xl flex-col gap-6">
      <div className="grid gap-1.5">
        <Label>Transform script</Label>
        <ScriptEditor value={scriptBody} onChange={setScriptBody} language="javascript" />
        <div className="flex gap-2">
          <Button type="button" size="sm" onClick={handleSave} disabled={updateSource.isPending}>
            Save
          </Button>
          <Button type="button" size="sm" variant="outline" onClick={handleClear} disabled={updateSource.isPending}>
            Clear
          </Button>
        </div>
      </div>

      <div className="grid gap-3 rounded-lg border p-4">
        <Label>Test run</Label>

        {!hasDeliveries ? (
          <p className="text-sm text-muted-foreground">
            You need at least one recorded delivery to test this script against.
          </p>
        ) : (
          <div className="flex items-center gap-2">
            <Select
              value={deliveryId ?? undefined}
              onValueChange={(value: string | null) => setDeliveryId(value)}
            >
              <SelectTrigger id="test-delivery" className="w-full">
                <SelectValue placeholder="Select a delivery" />
              </SelectTrigger>
              <SelectContent>
                {deliveries?.map((d) => (
                  <SelectItem key={d.id} value={d.id}>
                    {formatDeliveryLabel(d.id, d.received_at)}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
            <Button
              type="button"
              size="sm"
              onClick={() => testRun.mutate()}
              disabled={!deliveryId || testRun.isPending}
            >
              Run
            </Button>
          </div>
        )}

        {testRun.data && (
          testRun.data.error ? (
            <p className="text-sm text-destructive">{testRun.data.error}</p>
          ) : (
            <JsonViewer data={testRun.data.result} />
          )
        )}
      </div>
    </div>
  )
}
