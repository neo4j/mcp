//
// Copyright (c) "Neo4j"
// Neo4j Sweden AB [http://neo4j.com]

package server

import (
	"github.com/mark3labs/mcp-go/server"
	"github.com/neo4j/mcp/internal/tools"
	"github.com/neo4j/mcp/internal/tools/cypher"
	"github.com/neo4j/mcp/internal/tools/gds"
)

// registerTools registers all enabled MCP tools and adds them to the provided MCP server.
// Tools are filtered according to the server configuration. For example, when the read-only
// mode is enabled (e.g. via the NEO4J_READ_ONLY environment variable or the Config.ReadOnly flag),
// any tool that performs state mutation will be excluded; only tools annotated as read-only will be registered.
// Note: this read-only filtering relies on the tool annotation "readonly" (ReadOnlyHint). If the annotation
// is not defined or is set to false, the tool will be added (i.e., only tools with readonly=true are filtered in read-only mode).
func (s *Neo4jMCPServer) registerTools() error {
	filteredTools := s.getEnabledTools()
	s.MCPServer.AddTools(filteredTools...)
	return nil
}

type toolFilter func(tools []ToolDefinition) []ToolDefinition

type toolCategory int

const (
	cypherCategory toolCategory = 0
	gdsCategory    toolCategory = 1
)

type ToolDefinition struct {
	category   toolCategory
	definition server.ServerTool
	readonly   bool
}

func (s *Neo4jMCPServer) addGDSTools() {
	deps := &tools.ToolDependencies{
		DBService:        s.dbService,
		AnalyticsService: s.anService,
	}
	toolDefs := s.getAllToolsDefs(deps)
	toolDefinition := make([]server.ServerTool, 0)
	GDSTools := make([]ToolDefinition, 0, len(toolDefs))
	for _, t := range toolDefs {
		if t.category == gdsCategory {
			GDSTools = append(GDSTools, t)
		}
	}
	for _, toolDef := range GDSTools {
		toolDefinition = append(toolDefinition, toolDef.definition)
	}
	s.MCPServer.AddTools(toolDefinition...)
}

func (s *Neo4jMCPServer) getEnabledTools() []server.ServerTool {
	filters := make([]toolFilter, 0)

	// If read-only mode is enabled, expose only tools annotated as read-only.
	if s.config != nil && s.config.ReadOnly {
		filters = append(filters, filterWriteTools)
	}
	// If GDS is not installed, disable GDS tools.
	if !s.gdsInstalled {
		filters = append(filters, filterGDSTools)
	}
	deps := &tools.ToolDependencies{
		DBService:        s.dbService,
		AnalyticsService: s.anService,
	}
	toolDefs := s.getAllToolsDefs(deps)

	for _, filter := range filters {
		toolDefs = filter(toolDefs)
	}
	enabledTools := make([]server.ServerTool, 0)
	for _, toolDef := range toolDefs {
		enabledTools = append(enabledTools, toolDef.definition)
	}
	return enabledTools
}

func filterWriteTools(tools []ToolDefinition) []ToolDefinition {
	readOnlyTools := make([]ToolDefinition, 0, len(tools))
	for _, t := range tools {
		if t.readonly {
			readOnlyTools = append(readOnlyTools, t)
		}
	}
	return readOnlyTools
}

func filterGDSTools(tools []ToolDefinition) []ToolDefinition {
	nonGDSTools := make([]ToolDefinition, 0, len(tools))
	for _, t := range tools {
		if t.category != gdsCategory {
			nonGDSTools = append(nonGDSTools, t)
		}
	}
	return nonGDSTools
}

// getAllToolsDefs returns all available tools with their specs and handlers
func (s *Neo4jMCPServer) getAllToolsDefs(deps *tools.ToolDependencies) []ToolDefinition {

	return []ToolDefinition{
		{
			category: cypherCategory,
			definition: server.ServerTool{
				Tool:    cypher.GetSchemaSpec(),
				Handler: cypher.GetSchemaHandler(deps, s.config.SchemaSampleSize),
			},
			readonly: true,
		},
		{
			category: cypherCategory,
			definition: server.ServerTool{
				Tool:    cypher.ReadCypherSpec(),
				Handler: cypher.ReadCypherHandler(deps),
			},
			readonly: true,
		},
		{
			category: cypherCategory,
			definition: server.ServerTool{
				Tool:    cypher.WriteCypherSpec(),
				Handler: cypher.WriteCypherHandler(deps),
			},
			readonly: false,
		},
		// GDS Category/Section
		{
			category: gdsCategory,
			definition: server.ServerTool{
				Tool:    gds.ListGDSProceduresSpec(),
				Handler: gds.ListGdsProceduresHandler(deps),
			},
			readonly: true,
		},
		// Add other categories below...
	}
}
