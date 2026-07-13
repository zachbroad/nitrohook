import { useQueryState } from "nuqs"
import { AppShell } from "@/components/app-shell"
import { ErrorState } from "@/components/error-state"
import { ListPane } from "@/components/list-pane"
import { Input } from "@/components/ui/input"
import { useSources } from "@/lib/queries"
import { CreateSourceDialog } from "./create-source-dialog"

export function SourcesLayout() {
  const { data: sources = [], isLoading, isError, error, refetch } = useSources()
  const [search, setSearch] = useQueryState("q", { defaultValue: "" })

  const query = search.trim().toLowerCase()
  const filtered = query
    ? sources.filter((s) => s.name.toLowerCase().includes(query) || s.slug.toLowerCase().includes(query))
    : sources

  return (
    <AppShell
      list={
        <ListPane
          header={
            <div className="flex flex-col gap-2">
              <CreateSourceDialog />
              <Input
                placeholder="Search sources…"
                value={search}
                onChange={(e) => setSearch(e.target.value || null)}
              />
            </div>
          }
          error={isError
            ? <ErrorState title="Couldn't load sources" error={error} onRetry={() => refetch()} />
            : undefined}
          empty={isLoading ? "Loading…" : query ? "No matching sources" : "No sources yet"}
          items={filtered.map((s) => ({
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
