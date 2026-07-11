import * as React from "react"
import { NavLink } from "react-router-dom"
import { cn } from "@/lib/utils"

export interface ListItem { key: string; to: string; label: React.ReactNode }

export function ListPane({ header, items, empty }: {
  header?: React.ReactNode; items: ListItem[]; empty?: string
}) {
  return (
    <div className="flex flex-col">
      {header && <div className="p-2 border-b">{header}</div>}
      {items.length === 0 && <p className="p-4 text-sm text-muted-foreground">{empty ?? "Nothing yet"}</p>}
      <ul>
        {items.map((it) => (
          <li key={it.key}>
            <NavLink to={it.to} className={({ isActive }) =>
              cn("block px-4 py-2 text-sm border-b hover:bg-accent",
                 isActive && "bg-accent font-medium")}>
              {it.label}
            </NavLink>
          </li>
        ))}
      </ul>
    </div>
  )
}
