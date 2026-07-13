// Command mcp runs a Model Context Protocol server over stdio that exposes
// read-only access to NitroHook's sources, actions, and deliveries.
//
// Connect it from an MCP host (e.g. Claude Desktop) by launching this binary
// with DATABASE_URL pointed at the same Postgres that the API/worker use.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/joho/godotenv"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/zachbroad/nitrohook/internal/config"
	"github.com/zachbroad/nitrohook/internal/database"
	"github.com/zachbroad/nitrohook/internal/store"
)

type listSourcesInput struct{}

type listActionsInput struct {
	SourceSlug string `json:"source_slug" jsonschema:"slug of the source whose actions to list"`
}

type listDeliveriesInput struct {
	SourceSlug string `json:"source_slug,omitempty" jsonschema:"optional source slug to filter by"`
	Limit      int    `json:"limit,omitempty" jsonschema:"max rows to return (default 20, max 200)"`
}

func main() {
	// STDIO transport requires that nothing except JSON-RPC be written to stdout.
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, nil)))

	_ = godotenv.Load()
	cfg := config.Load()

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	pool, err := database.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	st := store.New(pool)

	server := mcp.NewServer(&mcp.Implementation{
		Name:    "nitrohook",
		Version: "0.1.0",
	}, nil)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_sources",
		Description: "List all NitroHook webhook sources.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, _ listSourcesInput) (*mcp.CallToolResult, any, error) {
		sources, err := st.Sources.List(ctx)
		if err != nil {
			return nil, nil, err
		}
		return jsonResult(sources)
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_actions",
		Description: "List actions configured for a source (by slug).",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in listActionsInput) (*mcp.CallToolResult, any, error) {
		src, err := st.Sources.GetBySlug(ctx, in.SourceSlug)
		if err != nil {
			return nil, nil, err
		}
		actions, err := st.Actions.List(ctx, src.ID)
		if err != nil {
			return nil, nil, err
		}
		return jsonResult(actions)
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_deliveries",
		Description: "List recent webhook deliveries, optionally filtered by source slug.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in listDeliveriesInput) (*mcp.CallToolResult, any, error) {
		limit := in.Limit
		if limit <= 0 {
			limit = 20
		}
		if limit > 200 {
			limit = 200
		}
		var slugPtr *string
		if in.SourceSlug != "" {
			slugPtr = &in.SourceSlug
		}
		deliveries, err := st.Deliveries.List(ctx, slugPtr, limit)
		if err != nil {
			return nil, nil, err
		}
		return jsonResult(deliveries)
	})

	slog.Info("nitrohook mcp server starting on stdio")
	if err := server.Run(ctx, &mcp.StdioTransport{}); err != nil {
		slog.Error("mcp server terminated", "error", err)
		os.Exit(1)
	}
}

func jsonResult(v any) (*mcp.CallToolResult, any, error) {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return nil, nil, fmt.Errorf("marshal result: %w", err)
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: string(b)}},
	}, nil, nil
}
