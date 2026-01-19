package server

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/neo4j/mcp/internal/analytics"
	"github.com/neo4j/mcp/internal/auth"
	"github.com/neo4j/mcp/internal/config"
	"github.com/neo4j/mcp/internal/database"
	"github.com/neo4j/neo4j-go-driver/v6/neo4j"
)

const (
	protocolHTTP                = "http"
	protocolHTTPS               = "https"
	serverHTTPShutdownTimeout   = 65 * time.Second  // Timeout for graceful shutdown (must exceed WriteTimeout to allow active requests to complete)
	serverHTTPReadHeaderTimeout = 5 * time.Second   // SECURITY: Maximum time to read request headers (prevents Slowloris attacks)
	serverHTTPReadTimeout       = 15 * time.Second  // SECURITY: Maximum time to read entire request including body (prevents slow-read attacks)
	serverHTTPWriteTimeout      = 60 * time.Second  // FUNCTIONALITY: Maximum time to write response (allows complex Neo4j queries and large result sets)
	serverHTTPIdleTimeout       = 120 * time.Second // PERFORMANCE: Maximum time to keep idle keep-alive connections open (improves connection reuse)
)

// Neo4jMCPServer represents the MCP server instance
type Neo4jMCPServer struct {
	MCPServer              *server.MCPServer
	httpServer             *http.Server
	httpServerReady        chan struct{}
	shutdownChan           chan struct{}
	config                 *config.Config
	dbService              database.Service
	version                string
	anService              analytics.Service
	gdsInstalled           bool
	connectionInfo         atomic.Value // stores analytics.ConnectionEventInfo; cached connection info (STDIO: set at startup, HTTP: cached after first tool call)
	httpConnectionInitSent sync.Once    // Ensures connection initialized event is sent only once in HTTP mode
}

// NewNeo4jMCPServer creates a new MCP server instance
// The config parameter is expected to be already validated
func NewNeo4jMCPServer(version string, cfg *config.Config, dbService database.Service, anService analytics.Service) *Neo4jMCPServer {
	neo4jServer := &Neo4jMCPServer{
		httpServerReady: make(chan struct{}),
		shutdownChan:    make(chan struct{}),
		config:          cfg,
		dbService:       dbService,
		version:         version,
		anService:       anService,
		gdsInstalled:    false,
	}

	// Configure hooks for tool call tracking
	hooks := neo4jServer.configureHooks()

	mcpServer := server.NewMCPServer(
		"neo4j-mcp",
		version,
		server.WithToolCapabilities(true),
		server.WithHooks(hooks),
		server.WithInstructions("This is the Neo4j official MCP server and can provide tool calling to interact with your Neo4j database,"+
			"by inferring the schema with tools like get-schema and executing arbitrary Cypher queries with read-cypher."),
	)

	neo4jServer.MCPServer = mcpServer
	return neo4jServer
}

// Start initializes and starts the MCP server
func (s *Neo4jMCPServer) Start() error {
	// Emit server startup event FIRST (before any DB operations) for both modes
	s.emitServerStartupEvent()

	// Verify requirements (includes DB queries) and emit connection event for STDIO mode
	// For HTTP mode: skip verification (no credentials yet)
	if s.config.TransportMode == config.TransportModeStdio {
		err := s.verifyRequirements()
		if err != nil {
			return err
		}
		// Emit connection initialized event after successful verification
		s.emitConnectionInitializedEvent(context.Background())
	}

	// Register tools
	if err := s.registerTools(); err != nil {
		return fmt.Errorf("failed to register tools: %w", err)
	}

	switch s.config.TransportMode {
	case config.TransportModeHTTP:
		return s.StartHTTPServer()
	case config.TransportModeStdio:
		slog.Info("Starting stdio server")
		return server.ServeStdio(s.MCPServer)
	default:
		return fmt.Errorf("unsupported transport mode: %s", s.config.TransportMode)
	}
}

// parseAllowedOrigins parses the allowed origins string into a slice of strings
func parseAllowedOrigins(allowedOriginsStr string) []string {
	if allowedOriginsStr == "" {
		return []string{}
	}

	if allowedOriginsStr == "*" {
		return []string{"*"}
	}
	origins := strings.Split(allowedOriginsStr, ",")
	allowedOrigins := make([]string, 0, len(origins))

	for _, origin := range origins {
		allowedOrigins = append(allowedOrigins, strings.TrimSpace(origin))
	}

	return allowedOrigins
}

// verifyRequirements check the Neo4j requirements:
// - A valid connection with a Neo4j instance.
// - The ability to perform a read query (database name is correctly defined).
// - Required plugin installed: APOC (specifically apoc.meta.schema as it's used for get-schema)
// - In case GDS is not installed a flag is set in the server and tools will be registered accordingly
// Note: In HTTP mode, these checks are skipped at startup since credentials come from per-request Basic Auth headers.
func (s *Neo4jMCPServer) verifyRequirements() error {
	// Skip verification in HTTP mode - credentials come from per-request Basic Auth headers
	if s.config.TransportMode == config.TransportModeHTTP {
		slog.Info("Skipping startup verification in HTTP mode (credentials required per-request)")
		return nil
	}

	err := s.dbService.VerifyConnectivity(context.Background())
	if err != nil {
		return fmt.Errorf("impossible to verify connectivity with the Neo4j instance: %w", err)
	}
	// Perform a dummy query to verify correctness of the connection, VerifyConnectivity is not exhaustive.
	records, err := s.dbService.ExecuteReadQuery(context.Background(), "RETURN 1 as first", map[string]any{})

	if err != nil {
		return fmt.Errorf("impossible to verify connectivity with the Neo4j instance: %w", err)
	}
	if len(records) != 1 || len(records[0].Values) != 1 {
		return fmt.Errorf("failed to verify connectivity with the Neo4j instance: unexpected response from test query")
	}
	one, ok := records[0].Values[0].(int64)
	if !ok || one != 1 {
		return fmt.Errorf("failed to verify connectivity with the Neo4j instance: unexpected response from test query")
	}
	// Check for apoc.meta.schema procedure
	checkApocMetaSchemaQuery := "SHOW PROCEDURES YIELD name WHERE name = 'apoc.meta.schema' RETURN count(name) > 0 AS apocMetaSchemaAvailable"

	// Check for apoc.meta.schema availability
	records, err = s.dbService.ExecuteReadQuery(context.Background(), checkApocMetaSchemaQuery, nil)
	if err != nil {
		return fmt.Errorf("failed to check for APOC availability: %w", err)
	}
	if len(records) != 1 || len(records[0].Values) != 1 {
		return fmt.Errorf("failed to verify APOC availability: unexpected response from test query")
	}
	apocMetaSchemaAvailable, ok := records[0].Values[0].(bool)
	if !ok || !apocMetaSchemaAvailable {
		return fmt.Errorf("please ensure the APOC plugin is installed and includes the 'meta' component")
	}
	// Call gds.version procedure to determine if GDS is installed
	records, err = s.dbService.ExecuteReadQuery(context.Background(), "RETURN gds.version() as gdsVersion", nil)
	if err != nil {
		// GDS is optional, so we log a warning and continue, assuming it's not installed.
		log.Print("Impossible to verify GDS installation.")
		s.gdsInstalled = false
		return nil
	}
	if len(records) == 1 && len(records[0].Values) == 1 {
		_, ok := records[0].Values[0].(string)
		if ok {
			s.gdsInstalled = true
		}
	}

	return nil
}

// emitServerStartupEvent emits the server startup event immediately with available info (no DB query)
func (s *Neo4jMCPServer) emitServerStartupEvent() {
	s.anService.EmitEvent(s.anService.NewStartupEvent(s.config.TransportMode))
}

// emitConnectionInitializedEvent emits the connection initialized event with DB information (STDIO mode only)
func (s *Neo4jMCPServer) emitConnectionInitializedEvent(ctx context.Context) {
	if !s.anService.IsEnabled() {
		return
	}

	records, err := s.dbService.ExecuteReadQuery(ctx, "CALL dbms.components()", map[string]any{})
	if err != nil {
		slog.Debug("Failed to collect connection metadata", "error", err.Error())
		return
	}

	connInfo := recordsToConnectionEventInfo(records)
	s.connectionInfo.Store(connInfo) // Cache for use in tool events
	s.anService.EmitEvent(s.anService.NewConnectionInitializedEvent(connInfo))
}

// collectConnectionInfo queries the database for connection information (HTTP mode - per tool call)
func (s *Neo4jMCPServer) collectConnectionInfo(ctx context.Context) analytics.ConnectionEventInfo {
	// In HTTP mode, verify auth is present in context before attempting DB query
	if s.config.TransportMode == config.TransportModeHTTP {
		if !auth.HasAuth(ctx) {
			slog.Error("Auth credentials not found in context for HTTP mode DB query",
				"operation", "collectConnectionInfo")
			return analytics.ConnectionEventInfo{
				Neo4jVersion:  "unknown",
				Edition:       "unknown",
				CypherVersion: []string{"unknown"},
			}
		}
	}

	records, err := s.dbService.ExecuteReadQuery(ctx, "CALL dbms.components()", map[string]any{})
	if err != nil {
		slog.Warn("Failed to collect connection info for tool event",
			"error", err.Error(),
			"mode", s.config.TransportMode)
		return analytics.ConnectionEventInfo{
			Neo4jVersion:  "unknown",
			Edition:       "unknown",
			CypherVersion: []string{"unknown"},
		}
	}

	return recordsToConnectionEventInfo(records)
}

// recordsToConnectionEventInfo converts dbms.components() records to ConnectionEventInfo
func recordsToConnectionEventInfo(records []*neo4j.Record) analytics.ConnectionEventInfo {
	// Default to "unknown" for all failure cases (empty records, malformed data, etc.)
	connInfo := analytics.ConnectionEventInfo{
		Neo4jVersion:  "unknown",
		Edition:       "unknown",
		CypherVersion: []string{"unknown"},
	}

	for _, record := range records {
		nameRaw, ok := record.Get("name")
		if !ok {
			slog.Debug("missing 'name' column in dbms.components record")
			continue
		}
		name, ok := nameRaw.(string)
		if !ok {
			slog.Debug("invalid 'name' type in dbms.components record")
			continue
		}

		editionRaw, ok := record.Get("edition")
		if !ok {
			slog.Debug("missing 'edition' column in dbms.components record")
			continue
		}
		edition, ok := editionRaw.(string)
		if !ok {
			slog.Debug("invalid 'edition' type in dbms.components record")
			continue
		}

		versionsRaw, ok := record.Get("versions")
		if !ok {
			slog.Debug("missing 'versions' column in dbms.components record")
			continue
		}
		versions, ok := versionsRaw.([]interface{})
		if !ok {
			slog.Debug("invalid 'versions' type in dbms.components record")
			continue
		}

		switch name {
		case "Neo4j Kernel":
			if len(versions) > 0 {
				if v, ok := versions[0].(string); ok {
					connInfo.Neo4jVersion = v
				}
			}
			connInfo.Edition = edition
		case "Cypher":
			var stringVersions []string
			for _, v := range versions {
				if s, ok := v.(string); ok {
					stringVersions = append(stringVersions, s)
				}
			}
			connInfo.CypherVersion = stringVersions
		}
	}
	return connInfo
}

// buildTLSConfig creates a TLS configuration with security best practices
// - Sets minimum TLS version to TLS 1.2 (allows TLS 1.3 negotiation)
// - Uses Go's default cipher suites (well-maintained and secure)
// - Compatible with self-signed and enterprise certificates
func (s *Neo4jMCPServer) buildTLSConfig() (*tls.Config, error) {
	// Load the certificate and key
	cert, err := tls.LoadX509KeyPair(s.config.HTTPTLSCertFile, s.config.HTTPTLSKeyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load TLS certificate and key: %w", err)
	}

	// Create TLS config with security best practices
	// MinVersion is set to TLS 1.2, which allows TLS 1.3 clients to negotiate higher versions
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
		// CipherSuites: nil (uses Go's default secure cipher suites)
		// PreferServerCipherSuites: deprecated in Go 1.17+ (server preference is always used for TLS 1.3)
	}

	return tlsConfig, nil
}

// Stop gracefully stops the HTTP server
func (s *Neo4jMCPServer) Stop(ctx context.Context) error {
	if s.httpServer != nil {
		slog.Info("Stopping HTTP server...")
		if err := s.httpServer.Shutdown(ctx); err != nil {
			slog.Error("Error shutting down HTTP server", "error", err)
			return err
		}
		// Signal the StartHTTPServer goroutine to exit
		close(s.shutdownChan)
		slog.Info("HTTP server stopped")
	}
	return nil
}

func (s *Neo4jMCPServer) StartHTTPServer() error {
	addr := fmt.Sprintf("%s:%s", s.config.HTTPHost, s.config.HTTPPort)
	protocol := protocolHTTP
	if s.config.HTTPTLSEnabled {
		protocol = protocolHTTPS
	}
	slog.Info("Starting HTTP server", "address", addr, "url", fmt.Sprintf("%s://%s", protocol, addr), "tls", s.config.HTTPTLSEnabled)

	// Create the StreamableHTTPServer - it serves on /mcp path by default
	mcpServerHTTP := server.NewStreamableHTTPServer(
		s.MCPServer,
		server.WithStateLess(true),
	)

	allowedOrigins := parseAllowedOrigins(s.config.HTTPAllowedOrigins)
	// Wrap handler with middleware and create HTTP server
	s.httpServer = &http.Server{
		Addr:    addr,
		Handler: s.chainMiddleware(allowedOrigins, mcpServerHTTP),
		// Timeouts optimized for stateless HTTP MCP requests
		ReadTimeout:       serverHTTPReadTimeout,
		WriteTimeout:      serverHTTPWriteTimeout,
		IdleTimeout:       serverHTTPIdleTimeout,
		ReadHeaderTimeout: serverHTTPReadHeaderTimeout,
	}

	// Configure TLS if enabled
	if s.config.HTTPTLSEnabled {
		tlsConfig, err := s.buildTLSConfig()
		if err != nil {
			return fmt.Errorf("failed to configure TLS: %w", err)
		}
		s.httpServer.TLSConfig = tlsConfig
		slog.Info("TLS configuration applied", "minVersion", "TLS 1.2 (allows TLS 1.3 negotiation)")
	}

	// Signal that httpServer is ready for reading
	close(s.httpServerReady)

	// Channel to receive server errors
	errChan := make(chan error, 1)
	go func() {
		var err error
		if s.config.HTTPTLSEnabled {
			// Use empty strings for cert/key files since they're already loaded in TLSConfig
			err = s.httpServer.ListenAndServeTLS("", "")
		} else {
			err = s.httpServer.ListenAndServe()
		}
		if err != nil && err != http.ErrServerClosed {
			errChan <- fmt.Errorf("HTTP server failed: %w", err)
		}
	}()

	// Channel to receive shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Block until we receive a signal, an error, or a shutdown request
	select {
	case sig := <-sigChan:
		slog.Info("Shutdown signal received", "signal", sig.String())
		shutdownCtx, cancel := context.WithTimeout(context.Background(), serverHTTPShutdownTimeout)
		defer cancel()
		if err := s.httpServer.Shutdown(shutdownCtx); err != nil {
			slog.Error("Error during server shutdown", "error", err)
			return err
		}
		close(s.shutdownChan)
		slog.Info("HTTP server stopped gracefully")
		return nil
	case err := <-errChan:
		return err
	case <-s.shutdownChan:
		// Server was stopped via Stop() method
		return nil
	}
}

// configureHooks sets up MCP SDK hooks for tool call tracking
func (s *Neo4jMCPServer) configureHooks() *server.Hooks {
	hooks := &server.Hooks{}
	hooks.AddAfterCallTool(s.handleToolCallComplete)
	return hooks
}

// handleToolCallComplete is called after every tool call completes
func (s *Neo4jMCPServer) handleToolCallComplete(ctx context.Context, _ any, request *mcp.CallToolRequest, result *mcp.CallToolResult) {
	if s.anService == nil || !s.anService.IsEnabled() {
		return
	}

	toolName := request.Params.Name
	success := !result.IsError

	// HTTP mode only: emit connection initialized event on first successful tool call
	// (STDIO mode emits this at startup instead)
	if s.config.TransportMode == config.TransportModeHTTP {
		// Check if connection info is already cached
		loaded := s.connectionInfo.Load()
		var needsCollection bool
		if loaded == nil {
			needsCollection = true
		} else if cachedInfo, ok := loaded.(analytics.ConnectionEventInfo); ok {
			needsCollection = cachedInfo.Neo4jVersion == ""
		}

		// Collect connection info once for CONNECTION_INITIALIZED event
		if needsCollection {
			connInfo := s.collectConnectionInfo(ctx)
			// Only cache if we got valid connection info (not empty, not "unknown")
			if connInfo.Neo4jVersion != "" && connInfo.Neo4jVersion != "unknown" {
				s.connectionInfo.Store(connInfo) // Cache for future tool calls
			}
		}

		// Emit connection initialized event only once per session if we have valid connection info
		if loaded := s.connectionInfo.Load(); loaded != nil {
			if connInfoSnapshot, ok := loaded.(analytics.ConnectionEventInfo); ok {
				hasValidConnectionInfo := connInfoSnapshot.Neo4jVersion != "" && connInfoSnapshot.Neo4jVersion != "unknown"
				if hasValidConnectionInfo {
					s.httpConnectionInitSent.Do(func() {
						s.anService.EmitEvent(s.anService.NewConnectionInitializedEvent(connInfoSnapshot))
					})
				}
			}
		}
	}

	// Emit tool event (connection info sent separately in CONNECTION_INITIALIZED event)
	s.anService.EmitEvent(s.anService.NewToolEvent(toolName, success))

	// Handle GDS events for cypher tools
	if toolName == "read-cypher" || toolName == "write-cypher" {
		s.emitGDSEventsIfNeeded(request)
	}
}

// emitGDSEventsIfNeeded checks if the cypher query contains GDS calls and emits appropriate events
func (s *Neo4jMCPServer) emitGDSEventsIfNeeded(request *mcp.CallToolRequest) {
	// Type assert Arguments to map[string]any
	args, ok := request.Params.Arguments.(map[string]any)
	if !ok {
		return
	}

	// Extract query from arguments
	queryRaw, ok := args["query"]
	if !ok {
		return
	}

	queryStr, ok := queryRaw.(string)
	if !ok {
		return
	}

	lowerQuery := strings.ToLower(queryStr)
	if strings.Contains(lowerQuery, "call gds.graph.project") {
		s.anService.EmitEvent(s.anService.NewGDSProjCreatedEvent())
	}
	if strings.Contains(lowerQuery, "call gds.graph.drop") {
		s.anService.EmitEvent(s.anService.NewGDSProjDropEvent())
	}
}
