import { AppShell } from "@/components/app-shell"
import { ListPane } from "@/components/list-pane"
import { useSources } from "@/lib/queries"
import { CreateSourceDialog } from "./create-source-dialog"

export function SourcesLayout() {
  const { data: sources = [], isLoading } = useSources()
  return (
    <AppShell
      list={
        <ListPane
          header={<CreateSourceDialog />}
          empty={isLoading ? "Loading…" : "No sources yet"}
          items={sources.map((s) => ({
            key: s.id,
            to: `/sources/${s.slug}`,
            label: (
              <span className="flex justify-between gap-2">
                <span className="truncate">{s.name}</span>
                <span className="text-xs text-muted-foreground">{s.mode}</span>
              </span>
            ),
          }))}
        />
      }
    />
  )
}
