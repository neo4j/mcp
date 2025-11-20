package server

import (
	"context"
	"log/slog"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/neo4j/mcp/internal/logger"
)

// onAfterSetLevelHook is called after the SetLevel method is executed. It updates the global logger level.
func (s *Neo4jMCPServer) onAfterSetLevelHook(_ context.Context, _ any, message *mcp.SetLevelRequest, _ *mcp.EmptyResult) {
	newLevel := string(message.Params.Level)
	logger.SetLevel(newLevel)
	slog.Info("Log level changed via MCP", "new_level", newLevel)
}
