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
	tools := getTools(deps)
	s.MCPServer.AddTools(tools...)
	return nil
}

// getTools returns the available tools with their specs and handlers.
// The list of tools returned is filtered based on the defined configuration.
// For instance, with "Config.ReadOnly" all tools that can perform state mutations will not be returned.
func getTools(deps *tools.ToolDependencies) []server.ServerTool {
	cypherTools := []server.ServerTool{
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
	}
	// GDS Category/Section
	gdsTools := []server.ServerTool{
		{
			Tool:    gds.ListGDSProceduresSpec(),
			Handler: gds.ListGdsProceduresHandler(deps),
		},
	}
	// Add other categories/modules below...

	allTools := append(cypherTools, gdsTools...)
	// filter out the returned tools:
	if deps.Config.ReadOnly == "true" {
		readOnlyTools := make([]server.ServerTool, 0, len(allTools))
		for _, tool := range allTools {
			if tool.Tool.Annotations.ReadOnlyHint != nil && *tool.Tool.Annotations.ReadOnlyHint == true {
				readOnlyTools = append(readOnlyTools, tool)
			}
		}
		return readOnlyTools
	}
	return allTools
}
