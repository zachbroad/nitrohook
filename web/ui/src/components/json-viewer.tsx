interface JsonViewerProps {
  data: unknown
}

/** Pretty-prints arbitrary JSON-serializable data in a scrollable, monospace block. */
export function JsonViewer({ data }: JsonViewerProps) {
  const text = data === undefined ? "undefined" : JSON.stringify(data, null, 2)

  return (
    <pre className="max-h-96 overflow-auto whitespace-pre-wrap break-words rounded-lg border bg-muted px-3 py-2 font-mono text-xs">
      {text}
    </pre>
  )
}
