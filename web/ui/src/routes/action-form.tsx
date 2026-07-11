import * as React from "react"
import { useForm, Controller, type FieldErrors } from "react-hook-form"
import { zodResolver } from "@hookform/resolvers/zod"
import { z } from "zod"

import { Button } from "@/components/ui/button"
import {
  Dialog,
  DialogClose,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Textarea } from "@/components/ui/textarea"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { useCreateAction, useUpdateAction } from "@/lib/queries"
import type { Action, ActionType } from "@/lib/types"

const webhookSchema = z.object({
  type: z.literal("webhook"),
  target_url: z.string().url("Must be a valid URL"),
  signing_secret: z.string().optional(),
  transform_script: z.string().optional(),
})
const javascriptSchema = z.object({
  type: z.literal("javascript"),
  script_body: z.string().min(1, "Script body is required"),
})
const slackSchema = z.object({
  type: z.literal("slack"),
  config: z.object({
    webhook_url: z.string().url("Must be a valid URL"),
    channel: z.string().optional(),
    username: z.string().optional(),
  }),
})
const smtpSchema = z.object({
  type: z.literal("smtp"),
  config: z.object({
    host: z.string().min(1, "Host is required"),
    port: z.coerce.number(),
    username: z.string().min(1, "Username is required"),
    password: z.string().min(1, "Password is required"),
    from: z.string().min(1, "From is required"),
    to: z.string().min(1, "To is required"),
    subject: z.string().min(1, "Subject is required"),
  }),
})
const twilioSchema = z.object({
  type: z.literal("twilio"),
  config: z.object({
    account_sid: z.string().min(1, "Account SID is required"),
    auth_token: z.string().min(1, "Auth token is required"),
    from: z.string().min(1, "From is required"),
    to: z.string().min(1, "To is required"),
    body_template: z.string().optional(),
  }),
})

export const actionSchema = z.discriminatedUnion("type", [
  webhookSchema,
  javascriptSchema,
  slackSchema,
  smtpSchema,
  twilioSchema,
])

export type ActionFormValues = z.output<typeof actionSchema>
/** Pre-coercion shape (e.g. smtp's `port` is a string in the input, number in the output). */
type ActionFormInput = z.input<typeof actionSchema>

/** Read a nested error message out of the union-typed FieldErrors without fighting TS's
 * inability to narrow error shapes across a discriminated union. */
function fieldError(errors: FieldErrors, path: string): string | undefined {
  const parts = path.split(".")
  let cur: unknown = errors
  for (const part of parts) {
    if (cur == null || typeof cur !== "object") return undefined
    cur = (cur as Record<string, unknown>)[part]
  }
  if (cur && typeof cur === "object" && "message" in cur) {
    return (cur as { message?: string }).message
  }
  return undefined
}

const TYPE_OPTIONS: { value: ActionType; label: string }[] = [
  { value: "webhook", label: "Webhook" },
  { value: "javascript", label: "JavaScript" },
  { value: "slack", label: "Slack" },
  { value: "smtp", label: "SMTP" },
  { value: "twilio", label: "Twilio" },
]

function defaultValuesForType(type: ActionType): ActionFormInput {
  switch (type) {
    case "webhook":
      return { type, target_url: "", signing_secret: "", transform_script: "" }
    case "javascript":
      return { type, script_body: "" }
    case "slack":
      return { type, config: { webhook_url: "", channel: "", username: "" } }
    case "smtp":
      return {
        type,
        config: { host: "", port: 587, username: "", password: "", from: "", to: "", subject: "" },
      }
    case "twilio":
      return { type, config: { account_sid: "", auth_token: "", from: "", to: "", body_template: "" } }
  }
}

/** Seed form defaults for edit mode from an existing Action row. */
export function defaultValuesFromAction(action: Action): ActionFormInput {
  const config = (action.config ?? {}) as Record<string, unknown>
  const str = (v: unknown) => (typeof v === "string" ? v : "")
  const num = (v: unknown) => (typeof v === "number" ? v : Number(v) || 0)

  switch (action.type) {
    case "webhook":
      return {
        type: "webhook",
        target_url: action.target_url ?? "",
        signing_secret: action.signing_secret ?? "",
        transform_script: action.transform_script ?? "",
      }
    case "javascript":
      return { type: "javascript", script_body: action.script_body ?? "" }
    case "slack":
      return {
        type: "slack",
        config: {
          webhook_url: str(config.webhook_url),
          channel: str(config.channel),
          username: str(config.username),
        },
      }
    case "smtp":
      return {
        type: "smtp",
        config: {
          host: str(config.host),
          port: num(config.port),
          username: str(config.username),
          password: str(config.password),
          from: str(config.from),
          to: str(config.to),
          subject: str(config.subject),
        },
      }
    case "twilio":
      return {
        type: "twilio",
        config: {
          account_sid: str(config.account_sid),
          auth_token: str(config.auth_token),
          from: str(config.from),
          to: str(config.to),
          body_template: str(config.body_template),
        },
      }
  }
}

/** Strip empty-string optional fields and build the exact API payload per type. */
export function toPayload(values: ActionFormValues): Record<string, unknown> {
  switch (values.type) {
    case "webhook": {
      const payload: Record<string, unknown> = { type: "webhook", target_url: values.target_url }
      if (values.signing_secret) payload.signing_secret = values.signing_secret
      if (values.transform_script) payload.transform_script = values.transform_script
      return payload
    }
    case "javascript":
      return { type: "javascript", script_body: values.script_body }
    case "slack": {
      const config: Record<string, unknown> = { webhook_url: values.config.webhook_url }
      if (values.config.channel) config.channel = values.config.channel
      if (values.config.username) config.username = values.config.username
      return { type: "slack", config }
    }
    case "smtp":
      return {
        type: "smtp",
        config: {
          host: values.config.host,
          port: values.config.port,
          username: values.config.username,
          password: values.config.password,
          from: values.config.from,
          to: values.config.to,
          subject: values.config.subject,
        },
      }
    case "twilio": {
      const config: Record<string, unknown> = {
        account_sid: values.config.account_sid,
        auth_token: values.config.auth_token,
        from: values.config.from,
        to: values.config.to,
      }
      if (values.config.body_template) config.body_template = values.config.body_template
      return { type: "twilio", config }
    }
  }
}

interface ActionFormProps {
  slug: string
  /** Present in edit mode; omit for create mode. */
  action?: Action
  /** Custom trigger element (e.g. an Edit icon button in a table row). Defaults to a "New action" button. */
  trigger?: React.ReactElement
}

export function ActionForm({ slug, action, trigger }: ActionFormProps) {
  const [open, setOpen] = React.useState(false)
  const isEdit = !!action
  const createAction = useCreateAction(slug)
  const updateAction = useUpdateAction(slug)

  const {
    register,
    handleSubmit,
    control,
    watch,
    reset,
    formState: { errors, isSubmitting },
  } = useForm<ActionFormInput, unknown, ActionFormValues>({
    resolver: zodResolver(actionSchema),
    defaultValues: action ? defaultValuesFromAction(action) : defaultValuesForType("webhook"),
  })

  const type = watch("type")

  const onOpenChange = (next: boolean) => {
    setOpen(next)
    if (next) {
      reset(action ? defaultValuesFromAction(action) : defaultValuesForType("webhook"))
    } else {
      reset()
    }
  }

  const onTypeChange = (next: ActionType) => {
    reset(defaultValuesForType(next))
  }

  const onSubmit = async (values: ActionFormValues) => {
    const payload = toPayload(values)
    try {
      if (isEdit && action) {
        await updateAction.mutateAsync({ id: action.id, body: payload })
      } else {
        await createAction.mutateAsync(payload as Parameters<typeof createAction.mutateAsync>[0])
      }
      setOpen(false)
      reset()
    } catch {
      // Failure is surfaced by the mutation's onError toast; keep the dialog
      // open so the user can retry, and swallow the rejection here so it
      // doesn't become an unhandled promise rejection.
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogTrigger render={trigger ?? <Button size="sm">New action</Button>} />
      <DialogContent>
        <form onSubmit={handleSubmit(onSubmit)}>
          <DialogHeader>
            <DialogTitle>{isEdit ? "Edit action" : "New action"}</DialogTitle>
          </DialogHeader>
          <div className="grid gap-4 py-2">
            <div className="grid gap-1.5">
              <Label htmlFor="action-type">Type</Label>
              <Controller
                name="type"
                control={control}
                render={({ field }) => (
                  <Select
                    value={field.value}
                    onValueChange={(value: ActionType | null) => {
                      if (!value) return
                      field.onChange(value)
                      onTypeChange(value)
                    }}
                  >
                    <SelectTrigger id="action-type" className="w-full">
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      {TYPE_OPTIONS.map((opt) => (
                        <SelectItem key={opt.value} value={opt.value}>
                          {opt.label}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                )}
              />
            </div>

            {type === "webhook" && (
              <>
                <div className="grid gap-1.5">
                  <Label htmlFor="target_url">Target URL</Label>
                  <Input
                    id="target_url"
                    placeholder="https://example.com/hooks"
                    {...register("target_url")}
                  />
                  {fieldError(errors, "target_url") && (
                    <p className="text-xs text-destructive">{fieldError(errors, "target_url")}</p>
                  )}
                </div>
                <div className="grid gap-1.5">
                  <Label htmlFor="signing_secret">Signing secret (optional)</Label>
                  <Input id="signing_secret" {...register("signing_secret")} />
                </div>
                <div className="grid gap-1.5">
                  <Label htmlFor="transform_script">Transform script (optional)</Label>
                  <Textarea id="transform_script" rows={4} {...register("transform_script")} />
                </div>
              </>
            )}

            {type === "javascript" && (
              <div className="grid gap-1.5">
                <Label htmlFor="script_body">Script body</Label>
                <Textarea id="script_body" rows={8} {...register("script_body")} />
                {fieldError(errors, "script_body") && (
                  <p className="text-xs text-destructive">{fieldError(errors, "script_body")}</p>
                )}
              </div>
            )}

            {type === "slack" && (
              <>
                <div className="grid gap-1.5">
                  <Label htmlFor="webhook_url">Webhook URL</Label>
                  <Input
                    id="webhook_url"
                    placeholder="https://hooks.slack.com/services/..."
                    {...register("config.webhook_url")}
                  />
                  {fieldError(errors, "config.webhook_url") && (
                    <p className="text-xs text-destructive">{fieldError(errors, "config.webhook_url")}</p>
                  )}
                </div>
                <div className="grid gap-1.5">
                  <Label htmlFor="channel">Channel (optional)</Label>
                  <Input id="channel" placeholder="#alerts" {...register("config.channel")} />
                </div>
                <div className="grid gap-1.5">
                  <Label htmlFor="username">Username (optional)</Label>
                  <Input id="username" {...register("config.username")} />
                </div>
              </>
            )}

            {type === "smtp" && (
              <>
                <div className="grid gap-1.5">
                  <Label htmlFor="host">Host</Label>
                  <Input id="host" placeholder="smtp.example.com" {...register("config.host")} />
                  {fieldError(errors, "config.host") && (
                    <p className="text-xs text-destructive">{fieldError(errors, "config.host")}</p>
                  )}
                </div>
                <div className="grid gap-1.5">
                  <Label htmlFor="port">Port</Label>
                  <Input id="port" type="number" {...register("config.port")} />
                  {fieldError(errors, "config.port") && (
                    <p className="text-xs text-destructive">{fieldError(errors, "config.port")}</p>
                  )}
                </div>
                <div className="grid gap-1.5">
                  <Label htmlFor="smtp-username">Username</Label>
                  <Input id="smtp-username" {...register("config.username")} />
                  {fieldError(errors, "config.username") && (
                    <p className="text-xs text-destructive">{fieldError(errors, "config.username")}</p>
                  )}
                </div>
                <div className="grid gap-1.5">
                  <Label htmlFor="password">Password</Label>
                  <Input id="password" type="password" {...register("config.password")} />
                  {fieldError(errors, "config.password") && (
                    <p className="text-xs text-destructive">{fieldError(errors, "config.password")}</p>
                  )}
                </div>
                <div className="grid gap-1.5">
                  <Label htmlFor="from">From</Label>
                  <Input id="from" placeholder="alerts@example.com" {...register("config.from")} />
                  {fieldError(errors, "config.from") && (
                    <p className="text-xs text-destructive">{fieldError(errors, "config.from")}</p>
                  )}
                </div>
                <div className="grid gap-1.5">
                  <Label htmlFor="to">To</Label>
                  <Input id="to" placeholder="team@example.com" {...register("config.to")} />
                  {fieldError(errors, "config.to") && (
                    <p className="text-xs text-destructive">{fieldError(errors, "config.to")}</p>
                  )}
                </div>
                <div className="grid gap-1.5">
                  <Label htmlFor="subject">Subject</Label>
                  <Input id="subject" {...register("config.subject")} />
                  {fieldError(errors, "config.subject") && (
                    <p className="text-xs text-destructive">{fieldError(errors, "config.subject")}</p>
                  )}
                </div>
              </>
            )}

            {type === "twilio" && (
              <>
                <div className="grid gap-1.5">
                  <Label htmlFor="account_sid">Account SID</Label>
                  <Input id="account_sid" {...register("config.account_sid")} />
                  {fieldError(errors, "config.account_sid") && (
                    <p className="text-xs text-destructive">{fieldError(errors, "config.account_sid")}</p>
                  )}
                </div>
                <div className="grid gap-1.5">
                  <Label htmlFor="auth_token">Auth token</Label>
                  <Input id="auth_token" type="password" {...register("config.auth_token")} />
                  {fieldError(errors, "config.auth_token") && (
                    <p className="text-xs text-destructive">{fieldError(errors, "config.auth_token")}</p>
                  )}
                </div>
                <div className="grid gap-1.5">
                  <Label htmlFor="twilio-from">From</Label>
                  <Input id="twilio-from" placeholder="+15551234567" {...register("config.from")} />
                  {fieldError(errors, "config.from") && (
                    <p className="text-xs text-destructive">{fieldError(errors, "config.from")}</p>
                  )}
                </div>
                <div className="grid gap-1.5">
                  <Label htmlFor="twilio-to">To</Label>
                  <Input id="twilio-to" placeholder="+15557654321" {...register("config.to")} />
                  {fieldError(errors, "config.to") && (
                    <p className="text-xs text-destructive">{fieldError(errors, "config.to")}</p>
                  )}
                </div>
                <div className="grid gap-1.5">
                  <Label htmlFor="body_template">Body template (optional)</Label>
                  <Textarea id="body_template" rows={3} {...register("config.body_template")} />
                </div>
              </>
            )}
          </div>
          <DialogFooter>
            <DialogClose render={<Button type="button" variant="outline" />}>Cancel</DialogClose>
            <Button type="submit" disabled={isSubmitting}>
              {isEdit ? "Save" : "Create"}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}
