// Copyright (c) "Neo4j"
// Neo4j Sweden AB [http://neo4j.com]

package server

import (
	"fmt"
	"log/slog"
	"slices"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/neo4j/mcp/internal/tools"
	"github.com/neo4j/mcp/internal/tools/cypher"
	"github.com/neo4j/mcp/internal/tools/gds"
)

// registerTools registers all enabled MCP tools and adds them to the provided MCP server.
// Tools are filtered according to the server configuration. For example, when the read-only
// mode is enabled (e.g. via the Config.ReadOnly flag, which can be set by the NEO4J_MCP_READ_ONLY environment variable or --read-only flag),
// any tool that performs state mutation will be excluded.
// Individual tools can also be selected via Config.Tools, which can be set by the NEO4J_MCP_TOOLS environment variable or -tools flag, with Config.ReadOnly taking precedence.
func (s *Neo4jMCPServer) registerTools() {
	deps := &tools.ToolDependencies{
		DBService:        s.dbService,
		AnalyticsService: s.anService,
	}

	var registered []string

	// Cypher section
	if toolName, ok := registerTool(s, cypher.GetSchemaSpec(), cypher.GetSchemaHandler(deps, s.config.SchemaSampleSize)); ok {
		registered = append(registered, toolName)
	}
	if toolName, ok := registerTool(s, cypher.ReadCypherSpec(), cypher.ReadCypherHandler(deps)); ok {
		registered = append(registered, toolName)
	}
	if toolName, ok := registerTool(s, cypher.WriteCypherSpec(), cypher.WriteCypherHandler(deps)); ok {
		registered = append(registered, toolName)
	}

	// GDS section
	if toolName, ok := registerTool(s, gds.ListGDSProceduresSpec(), gds.ListGdsProceduresHandler(deps)); ok {
		registered = append(registered, toolName)
	}

	slog.Info("Registered server tools", "count", len(registered), "tools", registered)
}

// registerTool applies filtering and, if the tool passes, adds it to s.MCPServer.
func registerTool[In, Out any](s *Neo4jMCPServer, tool *mcp.Tool, handler mcp.ToolHandlerFor[In, Out]) (string, bool) {
	if !slices.Contains(s.config.Tools, tool.Name) {
		return tool.Name, false
	}

	if s.config.ReadOnly && (tool.Annotations == nil || !tool.Annotations.ReadOnlyHint) {
		slog.Info(fmt.Sprintf("Ignoring tool '%s': not available in read-only mode", tool.Name))
		return tool.Name, false
	}

	mcp.AddTool(s.MCPServer, tool, withToolLogging(tool.Name, handler))
	s.toolsByName[tool.Name] = tool
	return tool.Name, true
}
