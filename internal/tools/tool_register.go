package tools

import (
	"github.com/mark3labs/mcp-go/server"
	"github.com/neo4j/mcp/internal/config"
	"github.com/neo4j/mcp/internal/database"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

// ToolDependencies contains all dependencies needed by tools
type ToolDependencies struct {
	Driver    *neo4j.DriverWithContext
	Config    *config.Config
	DBService database.DatabaseService
}

// GetAllTools returns all available tools with their specs and handlers
func GetAllTools(deps *ToolDependencies) []server.ServerTool {
	return []server.ServerTool{
		{
			Tool:    RunCypherSpec(),
			Handler: RunCypherHandler(deps),
		},
		{
			Tool:    GetSchemaSpec(),
			Handler: GetSchemaHandler(deps),
		},
	}
}

// RegisterAllTools registers all available MCP tools
func RegisterAllTools(mcpServer *server.MCPServer, deps *ToolDependencies) {
	tools := GetAllTools(deps)
	mcpServer.AddTools(tools...)
}
