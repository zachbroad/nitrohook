import * as React from "react"
import { Outlet } from "react-router-dom"
import { Header } from "@/components/header"

export function AppShell({ list }: { list: React.ReactNode }) {
  return (
    <div className="flex h-screen w-screen flex-col overflow-hidden">
      <Header />
      <div className="flex flex-1 overflow-hidden">
        <aside className="w-1/5 min-w-[220px] max-w-[360px] border-r flex flex-col">
          <div className="flex-1 overflow-y-auto">{list}</div>
        </aside>
        <main className="flex-1 overflow-y-auto"><Outlet /></main>
      </div>
    </div>
  )
}
