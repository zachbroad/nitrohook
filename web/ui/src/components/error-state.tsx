import { Button } from "@/components/ui/button"
import { describeApiError } from "@/lib/utils"

export function ErrorState({ title, error, onRetry }: {
  title: string; error: unknown; onRetry?: () => void
}) {
  return (
    <div className="flex h-full flex-col items-center justify-center gap-2 p-4 text-center">
      <p className="text-sm font-medium">{title}</p>
      <p className="text-sm text-muted-foreground">{describeApiError(error)}</p>
      {onRetry && (
        <Button type="button" variant="outline" size="sm" onClick={onRetry}>
          Retry
        </Button>
      )}
    </div>
  )
}
