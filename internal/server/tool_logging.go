// Copyright (c) "Neo4j"
// Neo4j Sweden AB [http://neo4j.com]

package server

import (
	"context"
	"log/slog"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/neo4j/mcp/internal/logger"
)

// Tool wrapper to handle shared logic across tools
// Intentionally keept the tool chain outside potential SDK-based hook to reduce future friction
// when moving to the official SDK
func withToolLogging[In, Out any](toolName string, handler mcp.ToolHandlerFor[In, Out]) mcp.ToolHandlerFor[In, Out] {
	return func(ctx context.Context, req *mcp.CallToolRequest, input In) (*mcp.CallToolResult, Out, error) {
		start := time.Now()

		slog.Debug("tool call started", append(logger.AppendRequestInfo(ctx), "tool", toolName)...)

		result, output, err := handler(ctx, req, input)
		durationMS := time.Since(start).Milliseconds()

		if err != nil {
			slog.Error("tool call completed", append(logger.AppendRequestInfo(ctx),
				"tool", toolName, "success", false, "duration_ms", durationMS, "error", err)...)
			return result, output, err
		}

		success := true
		if result != nil {
			success = !result.IsError
		}
		if success {
			slog.Info("tool call completed", append(logger.AppendRequestInfo(ctx),
				"tool", toolName, "success", true, "duration_ms", durationMS)...)
		} else {
			slog.Warn("tool call completed", append(logger.AppendRequestInfo(ctx),
				"tool", toolName, "success", false, "duration_ms", durationMS)...)
		}

		return result, output, nil
	}
}
