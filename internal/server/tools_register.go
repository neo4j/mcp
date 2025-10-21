package server

import (
	"github.com/mark3labs/mcp-go/server"
	"github.com/neo4j/mcp/internal/tools"
	"github.com/neo4j/mcp/internal/tools/cypher"
	"github.com/neo4j/mcp/internal/tools/gds"
)

// RegisterTools registers all available MCP tools
func (s *Neo4jMCPServer) RegisterTools() error {
	deps := &tools.ToolDependencies{
		Config:    s.config,
		DBService: s.dbService,
	}
	registerAllTools(s.MCPServer, deps)
	return nil
}

// RegisterAllTools registers all available MCP tools
func registerAllTools(mcpServer *server.MCPServer, deps *tools.ToolDependencies) {
	tools := getAllTools(deps)
	mcpServer.AddTools(tools...)
}

// getAllTools returns all available tools with their specs and handlers
func getAllTools(deps *tools.ToolDependencies) []server.ServerTool {
	return []server.ServerTool{
		{
			Tool:    cypher.GetSchemaSpec(),
			Handler: cypher.GetSchemaHandler(deps),
		},
		{
			Tool:    cypher.ReadCypherSpec(),
			Handler: cypher.ReadCypherHandler(deps),
		},
		{
			Tool:    cypher.WriteCypherSpec(),
			Handler: cypher.WriteCypherHandler(deps),
		},
		// GDS Category/Section
		{
			Tool:    gds.ListGDSProceduresSpec(),
			Handler: gds.ListGdsProceduresHandler(deps),
		},
		// Add other categories below...
	}
}
