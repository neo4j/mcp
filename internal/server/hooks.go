package server

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
)

// onAfterSetLevelHook is called after the SetLevel method is executed. It updates the server's logger level.
func (s *Neo4jMCPServer) onAfterSetLevelHook(_ context.Context, _ any, message *mcp.SetLevelRequest, _ *mcp.EmptyResult) {
	newLevel := string(message.Params.Level)
	s.log.SetLevel(newLevel)
	s.log.Info("Log level changed via MCP", "new_level", newLevel)
}
