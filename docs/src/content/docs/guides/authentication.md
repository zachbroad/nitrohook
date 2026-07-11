---
title: Authentication
description: Verify incoming webhooks with HMAC signatures, bearer tokens, or Ed25519.
---

Without authentication, anyone who knows a source's webhook URL can inject webhooks into your system. Enabling authentication on a source verifies each incoming request against a shared secret or public key before the webhook is stored or fanned out to actions. Failed requests return HTTP 401.

## Enabling authentication

On a source's Overview page, locate the **Authentication** card:

1. Check **Require authentication**
2. Select a preset from the dropdown (e.g. GitHub, Stripe, Slack)
3. Paste the shared secret or public key from your webhook provider
4. Click **Save**

From that point forward, all requests to the source's webhook URL must include the authentication headers or tokens expected by the chosen preset. Invalid or missing authentication returns HTTP 401 and the delivery is not stored.

## Supported presets

The following table shows each preset, the authentication scheme, and what credentials to provide:

| Preset | Scheme | Header(s) | Credentials | Notes |
|--------|--------|-----------|-------------|-------|
| **GitHub** | HMAC-SHA256 | `X-Hub-Signature-256` | Webhook secret | Signature format: `sha256={hex_digest}`. Signed over raw request body. |
| **Forgejo / Gitea** | HMAC-SHA256 | `X-Forgejo-Signature` | Webhook secret | Signed over raw request body. |
| **Stripe** | HMAC-SHA256 | `Stripe-Signature` | Signing secret | Format: `t={timestamp},v1={signature}`. Timestamp tolerance: 5 minutes. |
| **Slack** | HMAC-SHA256 | `X-Slack-Signature`, `X-Slack-Request-Timestamp` | Signing secret | Signed over `v0:{timestamp}:{body}`. Timestamp tolerance: 5 minutes. |
| **Shopify** | HMAC-SHA256 (base64) | `X-Shopify-Hmac-Sha256` | API secret key | Signature is base64-encoded. Signed over raw request body. |
| **Svix / Standard Webhooks** | HMAC-SHA256 (base64) | `webhook-id`, `webhook-timestamp`, `webhook-signature` | Signing secret | Signed over `{id}.{timestamp}.{body}`. Timestamp tolerance: 5 minutes. |
| **GitLab (token)** | Bearer token | `X-Gitlab-Token` | Webhook secret token | Not a signature — plain string equality. Secret travels on the wire. |
| **Discord (Ed25519)** | Ed25519 | `X-Signature-Ed25519`, `X-Signature-Timestamp` | Application public key (hex) | Signed over `{timestamp}{body}`. Asymmetric; paste the public key, not a secret. |

## Security best practices

- **Prefer signature schemes over bearer tokens.** HMAC and Ed25519 signatures provide tamper protection and do not expose the secret on the wire. Bearer tokens and plain-token schemes rely on secure transport and are only as safe as the infrastructure carrying them.
- **Secrets are stored plaintext at rest.** This is a known limitation; encryption is a planned enhancement.
- **Rotate secrets regularly** with your webhook provider, especially if you suspect compromise.

## Failures and monitoring

When authentication fails (missing headers, invalid signature, malformed request), the webhook is rejected with HTTP 401 before any delivery is stored or published to the fan-out queue.

Monitor authentication failures using the Prometheus counter:

```
nitrohook_webhook_auth_failures_total{source="<source_slug>", reason="<reason>"}
```

The `reason` label is one of: `missing_signature` (no signature/token header present), `bad_signature` (signature or token did not match), `missing_timestamp` (a timestamp-bound scheme was missing its timestamp), `timestamp` (timestamp outside the tolerance window), `unsupported` (misconfigured scheme), or `error` (other verification error).
