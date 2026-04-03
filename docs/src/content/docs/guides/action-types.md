---
title: Action Types
description: Configure webhook, Slack, SMTP, Twilio, and JavaScript actions.
---

Each action has a `type` and a `config` JSON object with type-specific settings.

## Webhook

Forwards the payload to an HTTP endpoint via POST.

```json
{
  "type": "webhook",
  "target_url": "https://example.com/hook",
  "is_active": true
}
```

The request includes the original (or transformed) payload as the body and headers. If a `signing_secret` is set on the action, the request includes an HMAC-SHA256 signature in the `X-Hook-Signature` header.

## Slack

Posts to a Slack channel via an incoming webhook URL.

```json
{
  "type": "slack",
  "config": {
    "webhook_url": "https://hooks.slack.com/services/T.../B.../xxx",
    "channel": "#alerts",
    "username": "nitrohook"
  },
  "is_active": true
}
```

| Field | Required | Description |
|-------|----------|-------------|
| `webhook_url` | Yes | Slack incoming webhook URL |
| `channel` | No | Override the default channel |
| `username` | No | Override the bot username |

## SMTP (Email)

Sends an email with the webhook payload as the body.

```json
{
  "type": "smtp",
  "config": {
    "host": "smtp.gmail.com",
    "port": 587,
    "username": "you@gmail.com",
    "password": "app-password",
    "from": "you@gmail.com",
    "to": "alerts@example.com",
    "subject": "Webhook Alert"
  },
  "is_active": true
}
```

| Field | Required | Description |
|-------|----------|-------------|
| `host` | Yes | SMTP server hostname |
| `port` | Yes | SMTP server port |
| `username` | No | SMTP auth username |
| `password` | No | SMTP auth password |
| `from` | Yes | Sender email address |
| `to` | Yes | Recipient email address |
| `subject` | No | Email subject (defaults to "Webhook delivery {id}") |

## Twilio (SMS)

Sends an SMS via the Twilio API.

```json
{
  "type": "twilio",
  "config": {
    "account_sid": "ACxxxxxxxx",
    "auth_token": "your-auth-token",
    "from": "+15551234567",
    "to": "+15559876543",
    "body_template": "Alert: new event received"
  },
  "is_active": true
}
```

| Field | Required | Description |
|-------|----------|-------------|
| `account_sid` | Yes | Twilio Account SID |
| `auth_token` | Yes | Twilio Auth Token |
| `from` | Yes | Sender phone number |
| `to` | Yes | Recipient phone number |
| `body_template` | No | Custom message body (defaults to payload summary, truncated to 1600 chars) |

## JavaScript

Runs a `process()` function in a sandboxed JS runtime. No external request is made — useful for validation, aggregation, or custom logic.

```json
{
  "type": "javascript",
  "script_body": "function process(event) {\n  return { ok: true };\n}",
  "is_active": true
}
```

The return value of `process()` is stored as the delivery attempt's response body. See [Scripting](/guides/scripting/) for details on the runtime.
