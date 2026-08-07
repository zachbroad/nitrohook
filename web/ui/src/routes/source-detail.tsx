import { NavLink, Outlet, useParams } from "react-router-dom"
import { cn } from "@/lib/utils"
import { useSource } from "@/lib/queries"
import { ApiError } from "@/lib/api"
import { ErrorState } from "@/components/error-state"

const TABS = [
  { path: "overview", label: "Overview" },
  { path: "actions", label: "Actions" },
  { path: "script", label: "Script" },
  { path: "events", label: "Events" },
] as const

export function SourceDetail() {
  const { slug = "" } = useParams()
  const { data: source, isLoading, isError, error, refetch } = useSource(slug)

  if (isLoading) {
    return <div className="p-4 text-sm text-muted-foreground">Loading…</div>
  }
  const notFound = error instanceof ApiError && error.status === 404
  if (notFound || (!source && !isError)) {
    return <div className="p-4 text-sm text-muted-foreground">Source not found</div>
  }
  if (isError || !source) {
    return <ErrorState title="Couldn't load source" error={error} onRetry={() => refetch()} />
  }

  return (
    <div className="flex h-full flex-col">
      <header className="border-b px-4 py-3">
        <h1 className="text-lg font-semibold leading-tight">{source.name}</h1>
        <p className="text-sm text-muted-foreground">{source.slug}</p>
      </header>
      <nav className="flex border-b px-4">
        {TABS.map((tab) => (
          <NavLink
            key={tab.path}
            to={`/sources/${slug}/${tab.path}`}
            className={({ isActive }) =>
              cn(
                "-mb-px border-b-2 px-3 py-2 text-sm font-medium",
                isActive
                  ? "border-primary text-foreground"
                  : "border-transparent text-muted-foreground hover:text-foreground"
              )
            }
          >
            {tab.label}
          </NavLink>
        ))}
      </nav>
      <div className="flex-1 overflow-y-auto p-4">
        <Outlet />
      </div>
    </div>
  )
}
