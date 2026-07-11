---
title: MCP server
description: Expose NitroHook sources, actions, and deliveries to LLM hosts via the Model Context Protocol.
---

NitroHook ships with an optional [Model Context Protocol](https://modelcontextprotocol.io) server at `cmd/mcp`. It lets an MCP-aware host (for example, Claude Desktop) inspect your webhook activity with read-only tool calls.

## What it exposes

The server speaks JSON-RPC over stdio and registers three tools:

| Tool | Input | Returns |
|---|---|---|
| `list_sources` | — | All sources, newest first |
| `list_actions` | `source_slug` | Actions configured for that source |
| `list_deliveries` | optional `source_slug`, optional `limit` (1–200, default 20) | Recent deliveries |

Each tool returns JSON as text content. The server connects directly to Postgres — Redis is not required.

## Build and run

```bash
make build            # produces bin/api, bin/worker, bin/mcp
DATABASE_URL=postgres://nitrohook:nitrohook@localhost:5432/nitrohook?sslmode=disable bin/mcp
```

For development, `make run-mcp` runs it under `go run`.

## Claude Desktop configuration

Add an entry to `~/Library/Application Support/Claude/claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "nitrohook": {
      "command": "/absolute/path/to/nitrohook/bin/mcp",
      "env": {
        "DATABASE_URL": "postgres://nitrohook:nitrohook@localhost:5432/nitrohook?sslmode=disable"
      }
    }
  }
}
```

Restart Claude Desktop. The `nitrohook` server should appear in the tools menu, and the three tools become available to the model.

## Implementation notes

- Built on the official [Go MCP SDK](https://github.com/modelcontextprotocol/go-sdk).
- Uses the `StdioTransport`; logs are routed to stderr so they do not corrupt the JSON-RPC stream.
- Tools are read-only — the MCP server is intentionally not a control plane. Use the REST API for mutations.
