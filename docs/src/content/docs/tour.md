---
title: Tour
description: A visual walkthrough of the nitrohook web UI.
---

A visual tour of the nitrohook web UI. For setup, see the [Quickstart](/getting-started/quickstart/).

## Sources

Each source has a public ingest URL, a mode (active or record), and a set of fan-out actions.

![Sources list](/screenshots/sources.png)

## Source overview

Drill into a source to see its ingest URL, signing secret, and recent activity at a glance.

![Source overview](/screenshots/source-overview.png)

## Actions

Webhook, Slack, SMTP, Twilio, or sandboxed JavaScript. Toggle active/inactive inline.

![Actions per source](/screenshots/actions.png)

## Transform scripts

A Monaco-powered editor for transform scripts. Mutate the payload, drop events by returning `null`, or filter `event.actions` to route selectively. Test against recorded payloads inline.

![Transform script editor](/screenshots/script-editor.png)

## Events

Every incoming webhook, with status (recorded / completed / failed) and idempotency key.

![Events list](/screenshots/deliveries.png)

## Event detail

Raw headers and payload alongside a per-attempt breakdown with HTTP status and errors.

![Event detail with attempts](/screenshots/delivery-detail.png)
