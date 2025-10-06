package server

import (
	"context"
	"fmt"
	"log"

	"github.com/mark3labs/mcp-go/server"
	"github.com/neo4j/mcp/internal/config"
	"github.com/neo4j/mcp/internal/database"
	"github.com/neo4j/mcp/internal/tools"
)

// Neo4jMCPServer represents the MCP server instance
type Neo4jMCPServer struct {
	mcpServer *server.MCPServer
	config    *config.Config
	dbService database.DatabaseService
	version   string
}

// NewNeo4jMCPServer creates a new MCP server instance
// The config parameter is expected to be already validated
func NewNeo4jMCPServer(version string, cfg *config.Config, dbService database.DatabaseService) *Neo4jMCPServer {
	mcpServer := server.NewMCPServer(
		"neo4j-mcp",
		version,
		server.WithToolCapabilities(true),
		server.WithInstructions("This is the Neo4j official MCP server and can provide tool calling to interact with your Neo4j database,"+
			"by inferring the schema with tools like get-schema and executing arbitrary Cypher queries with run-cypher."),
	)

	return &Neo4jMCPServer{
		mcpServer: mcpServer,
		config:    cfg,
		dbService: dbService,
		version:   version,
	}
}

// RegisterTools registers all available MCP tools
func (s *Neo4jMCPServer) RegisterTools() error {
	deps := &tools.ToolDependencies{
		Config:    s.config,
		DBService: s.dbService,
	}
	tools.RegisterAllTools(s.mcpServer, deps)
	return nil
}

// Start initializes and starts the MCP server using stdio transport
func (s *Neo4jMCPServer) Start(ctx context.Context) error {
	log.Println("Starting Neo4j MCP Server...")

	// Register tools
	if err := s.RegisterTools(); err != nil {
		return fmt.Errorf("failed to register tools: %w", err)
	}
	log.Println("Started Neo4j MCP Server. Now listening for input...")
	return server.ServeStdio(s.mcpServer)
}

// Stop gracefully stops the server
func (s *Neo4jMCPServer) Stop(ctx context.Context) error {
	log.Println("Stopping Neo4j MCP Server...")
	// Any cleanup logic for the server can go here
	return nil
}
