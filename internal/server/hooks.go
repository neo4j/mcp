package server

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
)

// onAfterSetLevelHook is called after the SetLevel method is executed. It updates the server's logger level.
func (s *Neo4jMCPServer) onAfterSetLevelHook(_ context.Context, id any, message *mcp.SetLevelRequest, result *mcp.EmptyResult) {
	newLevel := string(message.Params.Level) // Convert mcp.LoggingLevel to string
	s.log.SetLevel(newLevel)
	s.log.Info("Log level changed via MCP", "new_level", newLevel)
	// TODO remove debug log or make it conditional
	s.log.Debug("id and result details", "id", id, "result", result)
}
