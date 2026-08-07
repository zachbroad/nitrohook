import { WebhookIcon, CodeIcon, MessageSquareIcon, MailIcon, PhoneIcon } from "lucide-react"

import { Badge } from "@/components/ui/badge"
import type { ActionType } from "@/lib/types"

const CONFIG: Record<
  ActionType,
  { label: string; variant: "default" | "secondary" | "outline" | "destructive"; icon: typeof WebhookIcon; className?: string }
> = {
  webhook: { label: "Webhook", variant: "default", icon: WebhookIcon },
  javascript: {
    label: "JavaScript",
    variant: "secondary",
    icon: CodeIcon,
    className: "bg-amber-500/15 text-amber-700 dark:text-amber-400",
  },
  slack: {
    label: "Slack",
    variant: "secondary",
    icon: MessageSquareIcon,
    className: "bg-purple-500/15 text-purple-700 dark:text-purple-400",
  },
  smtp: {
    label: "Email",
    variant: "secondary",
    icon: MailIcon,
    className: "bg-sky-500/15 text-sky-700 dark:text-sky-400",
  },
  twilio: {
    label: "Twilio",
    variant: "secondary",
    icon: PhoneIcon,
    className: "bg-rose-500/15 text-rose-700 dark:text-rose-400",
  },
}

export function ActionTypeBadge({ type }: { type: ActionType }) {
  const { label, variant, icon: Icon, className } = CONFIG[type]
  return (
    <Badge variant={variant} className={className}>
      <Icon />
      {label}
    </Badge>
  )
}
