import { useParams } from "react-router-dom"
import { PencilIcon, TrashIcon } from "lucide-react"

import { Button } from "@/components/ui/button"
import { Switch } from "@/components/ui/switch"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import { ActionTypeBadge } from "@/components/action-type-badge"
import { useActions, useDeleteAction, useUpdateAction } from "@/lib/queries"
import type { Action } from "@/lib/types"
import { ActionForm } from "./action-form"

function summarize(action: Action): string {
  const config = (action.config ?? {}) as Record<string, unknown>
  const str = (v: unknown) => (typeof v === "string" ? v : "")
  switch (action.type) {
    case "webhook":
      return action.target_url ?? ""
    case "javascript":
      return "script"
    case "slack":
      return str(config.webhook_url)
    case "smtp":
      return str(config.to)
    case "twilio":
      return str(config.to)
    default:
      return ""
  }
}

export function SourceActions() {
  const { slug = "" } = useParams()
  const { data: actions = [], isLoading } = useActions(slug)
  const updateAction = useUpdateAction(slug)
  const deleteAction = useDeleteAction(slug)

  const handleDelete = (action: Action) => {
    if (!window.confirm(`Delete this ${action.type} action? This cannot be undone.`)) return
    deleteAction.mutate(action.id)
  }

  return (
    <div className="flex flex-col gap-4">
      <div className="flex items-center justify-between">
        <h2 className="text-sm font-medium text-muted-foreground">Actions</h2>
        <ActionForm slug={slug} />
      </div>

      {isLoading ? (
        <div className="p-4 text-sm text-muted-foreground">Loading…</div>
      ) : actions.length === 0 ? (
        <div className="p-4 text-sm text-muted-foreground">
          No actions yet. Create one to start fanning out deliveries.
        </div>
      ) : (
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Type</TableHead>
              <TableHead>Summary</TableHead>
              <TableHead>Active</TableHead>
              <TableHead className="text-right">Actions</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {actions.map((action) => (
              <TableRow key={action.id}>
                <TableCell>
                  <ActionTypeBadge type={action.type} />
                </TableCell>
                <TableCell className="max-w-xs truncate text-sm text-muted-foreground">
                  {summarize(action)}
                </TableCell>
                <TableCell>
                  <Switch
                    checked={action.is_active}
                    onCheckedChange={(checked) =>
                      updateAction.mutate({ id: action.id, body: { is_active: checked } })
                    }
                  />
                </TableCell>
                <TableCell className="text-right">
                  <div className="flex justify-end gap-1">
                    <ActionForm
                      slug={slug}
                      action={action}
                      trigger={
                        <Button variant="ghost" size="icon-sm" aria-label="Edit action">
                          <PencilIcon />
                        </Button>
                      }
                    />
                    <Button
                      variant="ghost"
                      size="icon-sm"
                      aria-label="Delete action"
                      onClick={() => handleDelete(action)}
                    >
                      <TrashIcon />
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
