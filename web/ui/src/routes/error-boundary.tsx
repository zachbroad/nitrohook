import { isRouteErrorResponse, useRouteError } from "react-router-dom"

function describeError(err: unknown): { heading: string; detail: string } {
  if (isRouteErrorResponse(err)) {
    return {
      heading: `${err.status} ${err.statusText}`,
      detail:
        typeof err.data === "string" ? err.data : "Something went wrong loading this page.",
    }
  }

  if (err instanceof Error) {
    return { heading: "Something went wrong", detail: err.message }
  }

  return { heading: "Something went wrong", detail: "An unexpected error occurred." }
}

export function RouteError() {
  const error = useRouteError()
  const { heading, detail } = describeError(error)

  return (
    <div className="h-full flex flex-col items-center justify-center gap-1 text-center text-muted-foreground">
      <p className="font-medium text-foreground">{heading}</p>
      <p className="text-sm">{detail}</p>
    </div>
  )
}
