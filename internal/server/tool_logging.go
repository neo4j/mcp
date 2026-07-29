// Copyright (c) "Neo4j"
// Neo4j Sweden AB [http://neo4j.com]

package server

import (
	"context"
	"log/slog"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/neo4j/mcp/internal/logger"
)

// Tool wrapper to handle shared logic across tools
// Intentionally keept the tool chain outside potential SDK-based hook to reduce future friction
// when moving to the official SDK
func withToolLogging(toolName string, handler server.ToolHandlerFunc) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		start := time.Now()

		slog.Info("tool call started", append(logger.AppendRequestInfo(ctx), "tool", toolName)...)

		result, err := handler(ctx, req)
		durationMS := time.Since(start).Milliseconds()

		if err != nil {
			slog.Error("tool call completed", append(logger.AppendRequestInfo(ctx),
				"tool", toolName, "success", false, "duration_ms", durationMS, "error", err)...)
			return result, err
		}

		success := true
		if result != nil {
			success = !result.IsError
		}
		slog.Info("tool call completed", append(logger.AppendRequestInfo(ctx),
			"tool", toolName, "success", success, "duration_ms", durationMS)...)

		return result, nil
	}
}
