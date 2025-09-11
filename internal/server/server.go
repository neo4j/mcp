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

// Neo4jMCPServer represents the MCP server instance
type Neo4jMCPServer struct {
	mcpServer *server.MCPServer
	config    *config.Config
	driver    *neo4j.DriverWithContext
}

// validateConfig validates the configuration and returns an error if invalid
func validateConfig(cfg *config.Config) error {
	if cfg == nil {
		return fmt.Errorf("configuration is required but was nil")
	}

	validations := []struct {
		value string
		name  string
	}{
		{cfg.URI, "Neo4j URI"},
		{cfg.Username, "Neo4j username"},
		{cfg.Password, "Neo4j password"},
		{cfg.Database, "Neo4j database name"},
	}

	for _, v := range validations {
		if v.value == "" {
			return fmt.Errorf("%s is required but was empty", v.name)
		}
	}

	return nil
}

// NewNeo4jMCPServer creates a new MCP server instance
func NewNeo4jMCPServer(cfg *config.Config) (*Neo4jMCPServer, error) {
	// Validate configuration
	if err := validateConfig(cfg); err != nil {
		log.Printf("Error in NewNeo4jMCPServer: %v", err)
		return nil, err
	}

	mcpServer := server.NewMCPServer(
		"neo4j-mcp",
		"0.0.1",
		server.WithToolCapabilities(true),
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
	log.Printf("Starting Neo4j MCP Server...")

	// Test the database connection
	if err := (*s.driver).VerifyConnectivity(ctx); err != nil {
		return fmt.Errorf("failed to verify database connectivity: %w", err)
	}

	// Register tools
	if err := s.RegisterTools(); err != nil {
		return fmt.Errorf("failed to register tools: %w", err)
	}

	return server.ServeStdio(s.mcpServer)
}

// Stop gracefully stops the server and closes the driver
func (s *Neo4jMCPServer) Stop(ctx context.Context) error {
	return (*s.driver).Close(ctx)
}
