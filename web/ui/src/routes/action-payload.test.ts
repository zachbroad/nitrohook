import { expect, test } from "vitest"
import { toPayload, defaultValuesFromAction, type ActionFormValues } from "./action-form"
import type { Action } from "@/lib/types"

const BASE_ACTION = {
  id: "act-1",
  source_id: "src-1",
  is_active: true,
  created_at: "2026-01-01T00:00:00Z",
  updated_at: "2026-01-01T00:00:00Z",
} satisfies Partial<Action>

// --- toPayload: field-name and nesting regression tests -------------------

test("toPayload: webhook produces top-level fields with no config key", () => {
  const values: ActionFormValues = {
    type: "webhook",
    target_url: "https://example.com/hooks",
    signing_secret: "shh",
    transform_script: "return payload;",
  }
  const payload = toPayload(values)

  expect(payload).toEqual({
    type: "webhook",
    target_url: "https://example.com/hooks",
    signing_secret: "shh",
    transform_script: "return payload;",
  })
  expect(payload).not.toHaveProperty("config")
})

test("toPayload: webhook omits empty optional fields", () => {
  const values: ActionFormValues = {
    type: "webhook",
    target_url: "https://example.com/hooks",
    signing_secret: "",
    transform_script: "",
  }
  const payload = toPayload(values)

  expect(payload).toEqual({ type: "webhook", target_url: "https://example.com/hooks" })
})

test("toPayload: javascript produces top-level script_body with no config key", () => {
  const values: ActionFormValues = { type: "javascript", script_body: "return payload;" }
  const payload = toPayload(values)

  expect(payload).toEqual({ type: "javascript", script_body: "return payload;" })
  expect(payload).not.toHaveProperty("config")
})

test("toPayload: slack nests fields under config", () => {
  const values: ActionFormValues = {
    type: "slack",
    config: { webhook_url: "https://hooks.slack.com/services/T/B/xyz", channel: "#alerts", username: "nitrohook" },
  }
  const payload = toPayload(values)

  expect(payload).toEqual({
    type: "slack",
    config: {
      webhook_url: "https://hooks.slack.com/services/T/B/xyz",
      channel: "#alerts",
      username: "nitrohook",
    },
  })
})

test("toPayload: slack omits empty optional fields from config", () => {
  const values: ActionFormValues = {
    type: "slack",
    config: { webhook_url: "https://hooks.slack.com/services/T/B/xyz", channel: "", username: "" },
  }
  const payload = toPayload(values)

  expect(payload).toEqual({
    type: "slack",
    config: { webhook_url: "https://hooks.slack.com/services/T/B/xyz" },
  })
})

test("toPayload: smtp nests all fields under config with port as a number", () => {
  const values: ActionFormValues = {
    type: "smtp",
    config: {
      host: "smtp.example.com",
      port: 587,
      username: "smtp-user",
      password: "smtp-pass",
      from: "alerts@example.com",
      to: "team@example.com",
      subject: "Webhook alert",
    },
  }
  const payload = toPayload(values)

  expect(payload).toEqual({
    type: "smtp",
    config: {
      host: "smtp.example.com",
      port: 587,
      username: "smtp-user",
      password: "smtp-pass",
      from: "alerts@example.com",
      to: "team@example.com",
      subject: "Webhook alert",
    },
  })
  const config = payload.config as Record<string, unknown>
  expect(typeof config.port).toBe("number")
})

test("toPayload: twilio nests required fields under config and includes body_template when set", () => {
  const values: ActionFormValues = {
    type: "twilio",
    config: {
      account_sid: "AC1234567890",
      auth_token: "auth-token-secret",
      from: "+15551234567",
      to: "+15557654321",
      body_template: "Alert: {{payload.message}}",
    },
  }
  const payload = toPayload(values)

  expect(payload).toEqual({
    type: "twilio",
    config: {
      account_sid: "AC1234567890",
      auth_token: "auth-token-secret",
      from: "+15551234567",
      to: "+15557654321",
      body_template: "Alert: {{payload.message}}",
    },
  })
})

test("toPayload: twilio omits body_template when empty", () => {
  const values: ActionFormValues = {
    type: "twilio",
    config: {
      account_sid: "AC1234567890",
      auth_token: "auth-token-secret",
      from: "+15551234567",
      to: "+15557654321",
      body_template: "",
    },
  }
  const payload = toPayload(values)

  expect(payload).toEqual({
    type: "twilio",
    config: {
      account_sid: "AC1234567890",
      auth_token: "auth-token-secret",
      from: "+15551234567",
      to: "+15557654321",
    },
  })
})

// --- defaultValuesFromAction: edit-mode seeding round-trip tests -----------

test("defaultValuesFromAction: webhook Action seeds top-level target_url and related fields", () => {
  const action: Action = {
    ...BASE_ACTION,
    type: "webhook",
    target_url: "https://example.com/hooks",
    signing_secret: "shh",
    transform_script: "return payload;",
  }
  const values = defaultValuesFromAction(action)

  expect(values).toEqual({
    type: "webhook",
    target_url: "https://example.com/hooks",
    signing_secret: "shh",
    transform_script: "return payload;",
  })
})

test("defaultValuesFromAction: slack Action with config.webhook_url seeds form's slack webhook_url", () => {
  const action: Action = {
    ...BASE_ACTION,
    type: "slack",
    config: { webhook_url: "https://hooks.slack.com/services/T/B/xyz", channel: "#alerts", username: "nitrohook" },
  }
  const values = defaultValuesFromAction(action)

  expect(values).toEqual({
    type: "slack",
    config: {
      webhook_url: "https://hooks.slack.com/services/T/B/xyz",
      channel: "#alerts",
      username: "nitrohook",
    },
  })
})

test("defaultValuesFromAction: smtp Action seeds nested config fields with numeric port", () => {
  const action: Action = {
    ...BASE_ACTION,
    type: "smtp",
    config: {
      host: "smtp.example.com",
      port: 2525,
      username: "smtp-user",
      password: "smtp-pass",
      from: "alerts@example.com",
      to: "team@example.com",
      subject: "Webhook alert",
    },
  }
  const values = defaultValuesFromAction(action)

  expect(values).toEqual({
    type: "smtp",
    config: {
      host: "smtp.example.com",
      port: 2525,
      username: "smtp-user",
      password: "smtp-pass",
      from: "alerts@example.com",
      to: "team@example.com",
      subject: "Webhook alert",
    },
  })
  expect(typeof (values as { config: { port: unknown } }).config.port).toBe("number")
})

test("defaultValuesFromAction: twilio Action seeds nested config fields", () => {
  const action: Action = {
    ...BASE_ACTION,
    type: "twilio",
    config: {
      account_sid: "AC1234567890",
      auth_token: "auth-token-secret",
      from: "+15551234567",
      to: "+15557654321",
      body_template: "Alert: {{payload.message}}",
    },
  }
  const values = defaultValuesFromAction(action)

  expect(values).toEqual({
    type: "twilio",
    config: {
      account_sid: "AC1234567890",
      auth_token: "auth-token-secret",
      from: "+15551234567",
      to: "+15557654321",
      body_template: "Alert: {{payload.message}}",
    },
  })
})

test("defaultValuesFromAction: javascript Action seeds top-level script_body", () => {
  const action: Action = { ...BASE_ACTION, type: "javascript", script_body: "return payload;" }
  const values = defaultValuesFromAction(action)

  expect(values).toEqual({ type: "javascript", script_body: "return payload;" })
})
