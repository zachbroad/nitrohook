---
title: Scripting
description: Transform, filter, and route webhooks with JavaScript.
---

nitrohook embeds a JavaScript runtime ([goja](https://github.com/nicholasgasior/goja)) that lets you transform payloads in-flight. Scripts run in a sandbox with a 500ms timeout and a 64KB size limit.

## Source-level transforms

A source transform runs on every delivery before actions are dispatched. Define a `transform` function:

```javascript
function transform(event) {
  // event.payload  — the webhook body (object)
  // event.headers  — request headers (object)
  // event.actions  — list of active actions [{id, target_url}]

  // Modify the payload
  event.payload.processed_at = new Date().toISOString();

  // Return the modified event
  return event;
}
```

### Filtering (dropping) events

Return `null` to drop a delivery entirely — no actions will fire:

```javascript
function transform(event) {
  // Only process "push" events
  if (event.headers["X-GitHub-Event"] !== "push") {
    return null;
  }
  return event;
}
```

### Routing to specific actions

Return a subset of `event.actions` to only dispatch to those actions:

```javascript
function transform(event) {
  // Only notify Slack for errors
  if (event.payload.level === "error") {
    return event; // all actions
  }

  // For non-errors, only forward to the first action
  event.actions = [event.actions[0]];
  return event;
}
```

## Action-level transforms

Each action can have its own transform script. This runs after the source transform and lets you reshape the payload for a specific destination. Define a `transform` function:

```javascript
function transform(event) {
  // event.payload — the (possibly already transformed) payload
  // event.headers — the (possibly already transformed) headers

  // Reshape for this specific action
  return {
    payload: {
      text: "Alert: " + event.payload.message,
    },
    headers: event.headers,
  };
}
```

Return `null` to skip this action for the current delivery:

```javascript
function transform(event) {
  // Skip this action for low-priority events
  if (event.payload.priority === "low") {
    return null;
  }
  return event;
}
```

## JavaScript action scripts

Actions of type `javascript` run a `process` function instead of dispatching to an external service:

```javascript
function process(event) {
  // Do something with event.payload and event.headers
  // The return value is stored as the attempt response body
  return { processed: true, items: event.payload.items.length };
}
```

## Limits

| Constraint | Value |
|-----------|-------|
| Max script size | 64 KB |
| Execution timeout | 500 ms |
| Runtime | goja (ES5.1 + some ES6) |
