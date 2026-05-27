// Copyright (c) "Neo4j"
// Neo4j Sweden AB [http://neo4j.com]

package server

import (
	"log/slog"
	"slices"

	"github.com/mark3labs/mcp-go/server"
	"github.com/neo4j/mcp/internal/tools"
	"github.com/neo4j/mcp/internal/tools/cypher"
	"github.com/neo4j/mcp/internal/tools/gds"
)

// registerTools registers all enabled MCP tools and adds them to the provided MCP server.
// Tools are filtered according to the server configuration. For example, when the read-only
// mode is enabled (e.g. via the Config.ReadOnly flag, which can be set by the NEO4J_READ_ONLY environment variable or --neo4j-read-only flag),
// any tool that performs state mutation will be excluded.
// Individual tools can also be selected via Config.Tools, which can be set by the NEO4J_TOOLS environment variable or --neo4j-tools flag, with Config.ReadOnly taking precedence.
func (s *Neo4jMCPServer) registerTools() {
	tools := s.getTools()
	s.MCPServer.AddTools(tools...)

	toolNames := make([]string, 0, len(tools))
	for _, tool := range tools {
		toolNames = append(toolNames, tool.Tool.Name)
	}
	slog.Info("Registered server tools", "count", len(toolNames), "tools", toolNames)
}

type ToolDefinition struct {
	definition server.ServerTool
	readonly   bool
}

func (s *Neo4jMCPServer) getTools() []server.ServerTool {
	deps := &tools.ToolDependencies{
		DBService:        s.dbService,
		AnalyticsService: s.anService,
	}
	toolsDefs := s.getAllToolsDefs(deps)
	serverTools := make([]server.ServerTool, 0, len(toolsDefs))
	for _, toolDef := range toolsDefs {
		if s.config.ReadOnly && !toolDef.readonly {
			continue
		}
		if !slices.Contains(s.config.Tools, toolDef.definition.Tool.Name) {
			continue
		}

		serverTools = append(serverTools, toolDef.definition)
	}
	return serverTools
}

// getAllToolsDefs returns all available tools with their specs and handlers
func (s *Neo4jMCPServer) getAllToolsDefs(deps *tools.ToolDependencies) []ToolDefinition {
	return []ToolDefinition{
		{
			definition: server.ServerTool{
				Tool:    cypher.GetSchemaSpec(),
				Handler: cypher.GetSchemaHandler(deps, s.config.SchemaSampleSize),
			},
			readonly: true,
		},
		{
			definition: server.ServerTool{
				Tool:    cypher.ReadCypherSpec(),
				Handler: cypher.ReadCypherHandler(deps),
			},
			readonly: true,
		},
		{
			definition: server.ServerTool{
				Tool:    cypher.WriteCypherSpec(),
				Handler: cypher.WriteCypherHandler(deps),
			},
			readonly: false,
		},
		// GDS Category/Section
		{
			definition: server.ServerTool{
				Tool:    gds.ListGDSProceduresSpec(),
				Handler: gds.ListGdsProceduresHandler(deps),
			},
			readonly: true,
		},
		// Add other categories below...
	}
}
