import CodeMirror, { type Extension } from "@uiw/react-codemirror"
import { javascript } from "@codemirror/lang-javascript"
import { json } from "@codemirror/lang-json"

export type ScriptEditorLanguage = "javascript" | "json"

const extensionsByLanguage: Record<ScriptEditorLanguage, Extension[]> = {
  javascript: [javascript()],
  json: [json()],
}

interface ScriptEditorProps {
  value: string
  onChange: (value: string) => void
  language: ScriptEditorLanguage
}

/** Controlled CodeMirror editor for JS transform scripts and JSON previews. */
export function ScriptEditor({ value, onChange, language }: ScriptEditorProps) {
  return (
    <CodeMirror
      value={value}
      onChange={onChange}
      extensions={extensionsByLanguage[language]}
      minHeight="200px"
      className="rounded-lg border text-sm"
    />
  )
}
