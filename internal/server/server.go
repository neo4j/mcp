package server

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/mark3labs/mcp-go/server"
	"github.com/neo4j/mcp/internal/config"
	"github.com/neo4j/mcp/internal/database"
	"github.com/neo4j/mcp/internal/tools"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

// Neo4jMCPServer represents the MCP server instance
type Neo4jMCPServer struct {
	mcpServer  *server.MCPServer
	httpServer *server.StreamableHTTPServer
	config     *config.Config
	driver     *neo4j.DriverWithContext
	version    string
}

// NewNeo4jMCPServer creates a new MCP server instance
// The config parameter is expected to be already validated
func NewNeo4jMCPServer(version string, cfg *config.Config) (*Neo4jMCPServer, error) {
	mcpServer := server.NewMCPServer(
		"neo4j-mcp",
		version,
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
		version:   version,
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

// Start initializes and starts the MCP server using the configured transport
func (s *Neo4jMCPServer) Start(ctx context.Context) error {
	log.Printf("Starting Neo4j MCP Server in %s mode...", s.config.TransportMode)

	// Test the database connection
	if err := (*s.driver).VerifyConnectivity(ctx); err != nil {
		return fmt.Errorf("failed to verify database connectivity: %w", err)
	}

	// Register tools
	if err := s.RegisterTools(); err != nil {
		return fmt.Errorf("failed to register tools: %w", err)
	}

	switch s.config.TransportMode {
	case config.TransportHTTP:
		return s.startHTTP()
	case config.TransportStdio:
		log.Println("Started Neo4j MCP Server. Now listening for input...")
		return server.ServeStdio(s.mcpServer)
	default:
		return fmt.Errorf("unsupported transport mode: %s", s.config.TransportMode)
	}
}

// startHTTP initializes and starts the HTTP server
func (s *Neo4jMCPServer) startHTTP() error {
	addr := fmt.Sprintf("%s:%s", s.config.HTTPHost, s.config.HTTPPort)

	// Create the StreamableHTTPServer with configuration
	s.httpServer = server.NewStreamableHTTPServer(
		s.mcpServer,
		server.WithEndpointPath(s.config.HTTPPath),
		server.WithStateLess(true),
	)

	log.Printf("Started Neo4j MCP HTTP Server on http://%s%s", addr, s.config.HTTPPath)
	log.Printf("Accepts both GET and POST requests")

	// Start the HTTP server
	httpServer := &http.Server{
		Addr:    addr,
		Handler: s.httpServer,
		ReadHeaderTimeout: time.Duration(10 * time.Second),
	}

	return httpServer.ListenAndServe()
}

// Stop gracefully stops the server and closes the driver
func (s *Neo4jMCPServer) Stop(ctx context.Context) error {
	return (*s.driver).Close(ctx)
}
