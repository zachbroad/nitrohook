import { NavLink } from "react-router-dom"
import { WebhookIcon } from "lucide-react"
import { cn } from "@/lib/utils"
import { ThemeToggle } from "@/components/theme-toggle"

const navItem = ({ isActive }: { isActive: boolean }) =>
  cn(
    "rounded-md px-3 py-1.5 text-sm font-medium transition-colors",
    isActive
      ? "bg-accent text-foreground"
      : "text-muted-foreground hover:bg-accent/50 hover:text-foreground",
  )

export function Header() {
  return (
    <header className="flex h-14 shrink-0 items-center gap-6 border-b px-4">
      <NavLink to="/" className="flex items-center gap-2" aria-label="NitroHook home">
        <span className="flex size-7 items-center justify-center rounded-md bg-primary text-primary-foreground">
          <WebhookIcon className="size-4" />
        </span>
        <span className="text-base font-semibold tracking-tight">NitroHook</span>
      </NavLink>

      <nav className="flex items-center gap-1">
        <NavLink to="/" end className={navItem}>
          Overview
        </NavLink>
        <NavLink to="/sources" className={navItem}>
          Sources
        </NavLink>
        <NavLink to="/deliveries" className={navItem}>
          Deliveries
        </NavLink>
      </nav>

      <div className="ml-auto">
        <ThemeToggle />
      </div>
    </header>
  )
}
