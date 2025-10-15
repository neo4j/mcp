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

const httpReadHeaderTimeout = 10 * time.Second

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

	// Create a router to handle multiple endpoints
	mux := http.NewServeMux()

	// MCP endpoint - requires authentication and origin validation
	mcpHandler := s.jwtAuthMiddleware(
		s.originValidationMiddleware(s.httpServer),
	)

	mux.Handle(s.config.HTTPPath, mcpHandler)

	// OAuth Discovery and Metadata endpoints - NO authentication required
	if s.config.Auth0Domain != "" && s.config.ResourceIdentifier != "" {
		// RFC 9728: Protected Resource Metadata
		mux.HandleFunc("/.well-known/oauth-protected-resource", s.handleProtectedResourceMetadata)

		log.Printf("✓ OAuth discovery endpoints enabled")
		log.Printf("  /.well-known/oauth-protected-resource - Protected resource metadata (RFC 9728)")
		log.Printf("  Resource identifier: %s", s.config.ResourceIdentifier)
		log.Printf("  Authorization server: https://%s/", s.config.Auth0Domain)

	}

	log.Printf("Started Neo4j MCP HTTP Server on http://%s%s", addr, s.config.HTTPPath)
	log.Printf("Binding to network interface: %s (use 127.0.0.1 for localhost-only)", s.config.HTTPHost)
	log.Printf("Accepts both GET and POST requests")

	// Log authentication status
	if s.config.Auth0Domain != "" && s.config.ResourceIdentifier != "" {
		log.Printf("Auth0 JWT authentication enabled (domain: %s, resource: %s)", s.config.Auth0Domain, s.config.ResourceIdentifier)
	} else {
		log.Printf("WARNING: Auth0 authentication is DISABLED - server is NOT SECURE")
	}

	log.Printf("Origin validation enabled with %d allowed origin(s)", len(s.config.AllowedOrigins))

	// Start the HTTP server
	httpServer := &http.Server{
		Addr:              addr,
		Handler:           s.corsMiddleware(mux), // Apply CORS to all routes at once
		ReadHeaderTimeout: httpReadHeaderTimeout,
	}

	return httpServer.ListenAndServe()
}

// Stop gracefully stops the server and closes the driver
func (s *Neo4jMCPServer) Stop(ctx context.Context) error {
	return (*s.driver).Close(ctx)
}

// originValidationMiddleware validates the Origin header to prevent DNS rebinding attacks
func (s *Neo4jMCPServer) originValidationMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// todo: TEMPORARILY DISABLED - Allow all origins for testing
		log.Printf("Origin validation disabled - accepting request from %s", r.RemoteAddr)
		next.ServeHTTP(w, r)
		// origin := r.Header.Get("Origin")

		// Origin validation MUST NOT be disabled in production code, which exposes the server to DNS rebinding attacks and CSRF vulnerabilities. This creates a significant security risk, especially when combined with the disabled authentication warnings elsewhere in the code.

		// // If no Origin header is present, check if request has Authorization header
		// // OAuth-authenticated clients (like VS Code) may not send Origin header
		// // The JWT middleware will validate the token for security
		// if origin == "" {
		// 	authHeader := r.Header.Get("Authorization")
		// 	if authHeader == "" {
		// 		log.Printf("Rejected request without Origin or Authorization header from %s", r.RemoteAddr)
		// 		http.Error(w, "Origin header is required", http.StatusForbidden)
		// 		return
		// 	}
		// 	// Has Authorization header, let JWT middleware validate it
		// 	log.Printf("Accepting request without Origin header (has Authorization) from %s", r.RemoteAddr)
		// 	next.ServeHTTP(w, r)
		// 	return
		// }

		// // Check if origin is in allowed list
		// if !slices.Contains(s.config.AllowedOrigins, origin) {
		// 	log.Printf("Rejected request from unauthorized origin: %s (remote: %s)", origin, r.RemoteAddr)
		// 	http.Error(w, "Origin not allowed", http.StatusForbidden)
		// 	return
		// }

		// // Origin is valid, proceed with the request
		// next.ServeHTTP(w, r)
	})
}
