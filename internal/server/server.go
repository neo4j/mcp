package server

import (
	"context"
	"fmt"
	"log"

	"github.com/mark3labs/mcp-go/server"
	"github.com/neo4j/mcp/internal/config"
	"github.com/neo4j/mcp/internal/database"
	"github.com/neo4j/mcp/internal/tools"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

// Version represents the current version of the neo4j-mcp server
const Version = "0.1.3"

// Neo4jMCPServer represents the MCP server instance
type Neo4jMCPServer struct {
	mcpServer *server.MCPServer
	config    *config.Config
	driver    *neo4j.DriverWithContext
	version   string
}

// NewNeo4jMCPServer creates a new MCP server instance
// The config parameter is expected to be already validated
func NewNeo4jMCPServer(cfg *config.Config) (*Neo4jMCPServer, error) {
	mcpServer := server.NewMCPServer(
		"neo4j-mcp",
		Version,
		server.WithToolCapabilities(true),
		server.WithInstructions("This is the Neo4j official MCP server and can provide tool calling to interact with your Neo4j database,"+
			"by inferring the schema with tools like get-schema and executing arbitrary Cypher queries with read-cypher."),
	)

	// Initialize Neo4j driver once
	driver, err := neo4j.NewDriverWithContext(cfg.URI, neo4j.BasicAuth(cfg.Username, cfg.Password, ""))
	if err != nil {
		wrappedErr := fmt.Errorf("failed to create Neo4j driver: %w", err)
		log.Printf("Error in NewNeo4jMCPServer: %v", wrappedErr)
		return nil, wrappedErr
	}

	return &Neo4jMCPServer{
		mcpServer: mcpServer,
		config:    cfg,
		driver:    &driver,
		version:   Version,
	}, nil
}

// RegisterTools registers all available MCP tools
func (s *Neo4jMCPServer) RegisterTools() error {
	// Create the database service
	dbService := database.NewNeo4jService(s.driver)

	deps := &tools.ToolDependencies{
		Config:    s.config,
		DBService: dbService,
	}
	tools.RegisterAllTools(s.mcpServer, deps)
	return nil
}

// Start initializes and starts the MCP server using stdio transport
func (s *Neo4jMCPServer) Start(ctx context.Context) error {
	log.Println("Starting Neo4j MCP Server...")

	// Test the database connection
	if err := (*s.driver).VerifyConnectivity(ctx); err != nil {
		return fmt.Errorf("failed to verify database connectivity: %w", err)
	}

	// Register tools
	if err := s.RegisterTools(); err != nil {
		return fmt.Errorf("failed to register tools: %w", err)
	}
	log.Println("Started Neo4j MCP Server. Now listening for input...")
	return server.ServeStdio(s.mcpServer)
}

// Stop gracefully stops the server and closes the driver
func (s *Neo4jMCPServer) Stop(ctx context.Context) error {
	return (*s.driver).Close(ctx)
}
