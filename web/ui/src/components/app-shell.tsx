import * as React from "react"
import { NavLink, Outlet } from "react-router-dom"
import { cn } from "@/lib/utils"

const tab = ({ isActive }: { isActive: boolean }) =>
  cn("flex-1 py-2 text-center text-sm font-medium border-b-2",
    isActive ? "border-primary text-foreground" : "border-transparent text-muted-foreground")

export function AppShell({ list }: { list: React.ReactNode }) {
  return (
    <div className="flex h-screen w-screen overflow-hidden">
      <aside className="w-1/5 min-w-[220px] max-w-[360px] border-r flex flex-col">
        <nav className="flex border-b">
          <NavLink to="/sources" className={tab}>Sources</NavLink>
          <NavLink to="/deliveries" className={tab}>Deliveries</NavLink>
        </nav>
        <div className="flex-1 overflow-y-auto">{list}</div>
      </aside>
      <main className="flex-1 overflow-y-auto"><Outlet /></main>
    </div>
  )
}
