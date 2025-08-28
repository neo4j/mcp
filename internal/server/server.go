package server

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/mark3labs/mcp-go/server"
	"github.com/neo4j/mcp/internal/config"
	"github.com/neo4j/mcp/internal/tools"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

// Neo4jMCPServer represents the MCP server instance
type Neo4jMCPServer struct {
	mcpServer *server.MCPServer
	config    *config.Config
	driver    *neo4j.DriverWithContext
}

// NewNeo4jMCPServer creates a new MCP server instance
func NewNeo4jMCPServer(cfg *config.Config) *Neo4jMCPServer {
	mcpServer := server.NewMCPServer(
		"neo4-mcp",
		"0.0.1",
	)

	// Initialize Neo4j driver once
	driver, err := neo4j.NewDriverWithContext(cfg.URI, neo4j.BasicAuth(cfg.Username, cfg.Password, ""))
	if err != nil {
		log.Fatalf("Error creating Neo4j driver: %v\n", err)
	}

	return &Neo4jMCPServer{
		mcpServer: mcpServer,
		config:    cfg,
		driver:    &driver,
	}
}

// RegisterTools registers all available MCP tools
func (s *Neo4jMCPServer) RegisterTools() {
	deps := &tools.ToolDependencies{
		Driver: s.driver,
		Config: s.config,
	}
	tools.RegisterAllTools(s.mcpServer, deps)
}

// Start starts the MCP server with stdio transport
func (s *Neo4jMCPServer) Start() error {
	fmt.Fprintf(os.Stderr, "Starting Cypher MCP Server running on stdio\n")

	// Start the server with stdio transport (this is blocking)
	if err := server.ServeStdio(s.mcpServer); err != nil {
		return fmt.Errorf("server error: %w", err)
	}

	return nil
}

// Stop gracefully stops the server and closes the driver
func (s *Neo4jMCPServer) Stop() error {
	fmt.Print("Gracefully stop the server")
	ctx := context.Background()
	return (*s.driver).Close(ctx)
}
