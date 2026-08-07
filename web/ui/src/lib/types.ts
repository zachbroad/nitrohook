export interface SourceAuthConfig {
  enabled: boolean; scheme: string; preset?: string;
  public_key?: string; has_secret: boolean;
}
export interface AuthPreset {
  name: string; label: string; needs_secret: boolean; needs_public_key: boolean;
}
export interface Source {
  id: string; name: string; slug: string; mode: "active" | "record";
  script_body?: string | null; auth_config?: SourceAuthConfig | null;
  created_at: string; updated_at: string;
}
export type ActionType = "webhook" | "javascript" | "slack" | "smtp" | "twilio";
export interface Action {
  id: string; source_id: string; type: ActionType;
  target_url?: string | null; script_body?: string | null; signing_secret?: string | null;
  config?: unknown; transform_script?: string | null; is_active: boolean;
  created_at: string; updated_at: string;
}
export type DeliveryStatus = "pending" | "processing" | "completed" | "failed" | "recorded";
export interface Delivery {
  id: string; source_id: string; idempotency_key: string;
  headers: unknown; payload: unknown; status: DeliveryStatus; received_at: string;
  transformed_payload?: unknown; transformed_headers?: unknown; retry_count: number;
}
export type AttemptStatus = "pending" | "success" | "failed";
export interface DeliveryAttempt {
  id: string; delivery_id: string; action_id: string; attempt_number: number;
  status: AttemptStatus; response_status?: number | null; response_body?: string | null;
  error_message?: string | null; next_retry_at?: string | null; created_at: string;
}
export interface TransformResult {
  payload: Record<string, unknown>; headers: Record<string, string>;
  actions: { id: string; target_url: string }[]; dropped: boolean;
}
export interface ScriptTestResult { result: TransformResult | null; error: string | null }
