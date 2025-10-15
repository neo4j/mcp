package tools

import (
	"github.com/mark3labs/mcp-go/server"
	"github.com/neo4j/mcp/internal/config"
	"github.com/neo4j/mcp/internal/database"
)

// ToolDependencies contains all dependencies needed by tools
type ToolDependencies struct {
	Config    *config.Config
	DBService database.Service
}

// GetAllTools returns all available tools with their specs and handlers
func GetAllTools(deps *ToolDependencies) []server.ServerTool {
	return []server.ServerTool{
		{
			Tool:    GetSchemaSpec(),
			Handler: GetSchemaHandler(deps),
		},
		{
			Tool:    ReadCypherSpec(),
			Handler: ReadCypherHandler(deps),
		},
		{
			Tool:    WriteCypherSpec(),
			Handler: WriteCypherHandler(deps),
		},
		{
			Tool:    GetGDSFunctionDetailsSpec(),
			Handler: GetGDSFunctionDetailsHandler(deps),
		},
		{
			Tool:    SummaryOfGDSFunctionsSpec(),
			Handler: SummaryOfGDSFunctionsHandler(deps),
		},
		{
			Tool:    ExecuteGDSFunctionSpec(),
			Handler: ExecuteGDSFunctionHandler(deps),
		},
	}
}

// RegisterAllTools registers all available MCP tools
func RegisterAllTools(mcpServer *server.MCPServer, deps *ToolDependencies) {
	tools := GetAllTools(deps)
	mcpServer.AddTools(tools...)
}
