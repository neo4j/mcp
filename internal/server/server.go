package server

import (
	"fmt"

	"github.com/mark3labs/mcp-go/server"
	"github.com/neo4j/mcp/internal/analytics"
	"github.com/neo4j/mcp/internal/config"
	"github.com/neo4j/mcp/internal/database"
	"github.com/neo4j/mcp/internal/logger"
)

// Neo4jMCPServer represents the MCP server instance
type Neo4jMCPServer struct {
	MCPServer *server.MCPServer
	config    *config.Config
	dbService database.Service
	version   string
	anService analytics.Service
	log       *logger.Service
}

// NewNeo4jMCPServer creates a new MCP server instance
// The config parameter is expected to be already validated
func NewNeo4jMCPServer(version string, cfg *config.Config, dbService database.Service, anService analytics.Service, log *logger.Service) *Neo4jMCPServer {
	// Create the server struct first, so we can reference it in the hooks.
	srv := &Neo4jMCPServer{
		config:    cfg,
		dbService: dbService,
		version:   version,
		log:       log,
		anService: anService,
	}

	mcpServer := server.NewMCPServer(
		"neo4j-mcp",
		version,
		server.WithToolCapabilities(true),
		server.WithInstructions("This is the Neo4j official MCP server and can provide tool calling to interact with your Neo4j database,"+
			"by inferring the schema with tools like get-schema and executing arbitrary Cypher queries with read-cypher."),
		server.WithLogging(),
		server.WithHooks(&server.Hooks{
			OnAfterSetLevel: []server.OnAfterSetLevelFunc{srv.onAfterSetLevelHook},
		}),
	)

	// Assign the created mcpServer to our struct.
	srv.MCPServer = mcpServer

	return srv
}

// Start initializes and starts the MCP server using stdio transport
func (s *Neo4jMCPServer) Start() error {
	s.log.Info("Starting Neo4j MCP Server...")

	// track startup event
	s.anService.EmitEvent(s.anService.NewStartupEvent())

	// Register tools
	if err := s.RegisterTools(); err != nil {
		return fmt.Errorf("failed to register tools: %w", err)
	}
	s.log.Info("Started Neo4j MCP Server. Now listening for input...")
	// Note: ServeStdio handles its own signal management for graceful shutdown
	return server.ServeStdio(s.MCPServer)
}

// Stop gracefully stops the server
func (s *Neo4jMCPServer) Stop() error {
	s.log.Info("Stopping Neo4j MCP Server...")
	// Currently no cleanup needed - the MCP server handles its own lifecycle
	// Database service cleanup is handled by the caller (main.go)
	return nil
}
