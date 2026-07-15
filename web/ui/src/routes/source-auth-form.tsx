import { useState } from "react"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { Switch } from "@/components/ui/switch"
import { useAuthPresets, useUpdateSourceAuth } from "@/lib/queries"
import type { Source } from "@/lib/types"

export function SourceAuthForm({ source }: { source: Source }) {
  const { data: presets } = useAuthPresets()
  const updateAuth = useUpdateSourceAuth(source.slug)
  const auth = source.auth_config

  const [enabled, setEnabled] = useState(auth?.enabled ?? false)
  const [preset, setPreset] = useState(auth?.preset ?? "github")
  const [secret, setSecret] = useState("")
  const [publicKey, setPublicKey] = useState(auth?.public_key ?? "")
  const [error, setError] = useState<string | null>(null)

  const presetInfo = presets?.find((p) => p.name === preset)
  const presetItems = (presets ?? []).map((p) => ({ value: p.name, label: p.label }))
  const hasSavedSecret = (auth?.has_secret ?? false) && auth?.preset === preset

  const save = () => {
    setError(null)
    if (!enabled) {
      updateAuth.mutate({ enabled: false })
      return
    }
    // The server never echoes secrets back, so every save must re-send one.
    if (presetInfo?.needs_secret && secret.trim() === "") {
      setError(
        hasSavedSecret
          ? "Re-enter the secret to save changes — it is never displayed after saving."
          : "A secret is required for this provider.",
      )
      return
    }
    if (presetInfo?.needs_public_key && publicKey.trim() === "") {
      setError("A public key is required for this provider.")
      return
    }
    updateAuth.mutate(
      { enabled: true, preset, secret: secret.trim(), public_key: publicKey.trim() },
      { onSuccess: () => setSecret("") },
    )
  }

  return (
    <div className="grid gap-3">
      <div className="grid gap-1.5">
        <Label htmlFor="source-auth-switch">Authentication</Label>
        <div className="flex items-center gap-3">
          <Switch
            id="source-auth-switch"
            checked={enabled}
            onCheckedChange={(v) => {
              setEnabled(v)
              setError(null)
            }}
          />
          <span className="text-sm">
            {enabled
              ? "Incoming webhooks must pass verification before they are stored"
              : "Incoming webhooks are accepted without verification"}
          </span>
        </div>
      </div>

      {enabled && (
        <>
          <div className="grid gap-1.5">
            <Label htmlFor="source-auth-preset">Provider</Label>
            <Select
              value={preset}
              items={presetItems}
              onValueChange={(v: string | null) => {
                if (!v) return
                setPreset(v)
                setError(null)
              }}
            >
              <SelectTrigger id="source-auth-preset" className="w-full">
                <SelectValue placeholder="Select a provider" />
              </SelectTrigger>
              <SelectContent>
                {(presets ?? []).map((p) => (
                  <SelectItem key={p.name} value={p.name}>
                    {p.label}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>

          {(presetInfo?.needs_secret ?? true) && (
            <div className="grid gap-1.5">
              <Label htmlFor="source-auth-secret">Shared secret / token</Label>
              <Input
                id="source-auth-secret"
                type="password"
                autoComplete="off"
                placeholder={
                  hasSavedSecret
                    ? "A secret is saved — re-enter it to make changes"
                    : "Paste the signing secret from the sending platform"
                }
                value={secret}
                onChange={(e) => setSecret(e.target.value)}
              />
            </div>
          )}

          {presetInfo?.needs_public_key && (
            <div className="grid gap-1.5">
              <Label htmlFor="source-auth-public-key">Public key</Label>
              <Input
                id="source-auth-public-key"
                autoComplete="off"
                placeholder="Hex-encoded Ed25519 public key"
                value={publicKey}
                onChange={(e) => setPublicKey(e.target.value)}
              />
            </div>
          )}

          <p className="text-xs text-muted-foreground">
            HMAC and Ed25519 verify a signature without exposing the secret on the
            wire. Bearer/token schemes send the secret with every request — prefer
            a signature scheme when the sender supports one.
          </p>
        </>
      )}

      {error && <p className="text-xs text-destructive">{error}</p>}

      <div>
        <Button
          type="button"
          size="sm"
          onClick={save}
          disabled={updateAuth.isPending}
        >
          Save authentication
        </Button>
      </div>
    </div>
  )
}
