// Copyright (c) "Neo4j"
// Neo4j Sweden AB [http://neo4j.com]

package server

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"slices"
	"strings"
	"syscall"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/neo4j/mcp/internal/analytics"
	"github.com/neo4j/mcp/internal/config"
	"github.com/neo4j/mcp/internal/database"
	"github.com/neo4j/mcp/internal/logger"
	"github.com/neo4j/mcp/internal/mcpcontext"
	"github.com/neo4j/mcp/internal/tools"
	"github.com/neo4j/neo4j-go-driver/v6/neo4j"
)

const (
	protocolHTTP                = "http"
	protocolHTTPS               = "https"
	serverHTTPReadHeaderTimeout = 5 * time.Second   // SECURITY: Maximum time to read request headers (prevents Slowloris attacks)
	serverHTTPReadTimeout       = 15 * time.Second  // SECURITY: Maximum time to read entire request including body (prevents slow-read attacks)
	serverHTTPIdleTimeout       = 120 * time.Second // PERFORMANCE: Maximum time to keep idle keep-alive connections open (improves connection reuse)
	httpWriteTimeoutGrace       = 5 * time.Second   // Buffer added to WriteTimeout so timeout error responses can be written
	httpShutdownGrace           = 5 * time.Second   // Buffer added to shutdown timeout so active requests can complete
)

// Neo4jMCPServer represents the MCP server instance
type Neo4jMCPServer struct {
	MCPServer       *mcp.Server
	httpServer      *http.Server
	HTTPServerReady chan struct{}
	shutdownChan    chan struct{}
	config          *config.Config
	dbService       database.Service
	version         string
	anService       analytics.Service
	uriResolver     URIResolver
	driverRegistry  database.DriverRegistry
	// toolsByName holds every registered tool's spec, keyed by name, so the
	// tools/call middleware can check a tool's ReadOnlyHint without relying on
	// SDK-internal registry introspection (no public equivalent in go-sdk).
	toolsByName map[string]*mcp.Tool
}

// NewNeo4jMCPServer creates a new MCP server instance
// The config parameter is expected to be already validated
func NewNeo4jMCPServer(version string, cfg *config.Config, dbService database.Service, anService analytics.Service) *Neo4jMCPServer {
	neo4jServer := &Neo4jMCPServer{
		HTTPServerReady: make(chan struct{}),
		shutdownChan:    make(chan struct{}),
		config:          cfg,
		dbService:       dbService,
		version:         version,
		anService:       anService,
		toolsByName:     make(map[string]*mcp.Tool),
	}

	if cfg.TransportMode == config.TransportModeHTTP {
		neo4jServer.uriResolver = &HeaderURIResolver{}
		neo4jServer.driverRegistry = &database.PerRequestDriverRegistry{}
	}

	mcpServer := mcp.NewServer(
		&mcp.Implementation{Name: "neo4j-mcp", Version: version},
		&mcp.ServerOptions{
			Instructions: "This is the Neo4j official MCP server and can provide tool calling to interact with your Neo4j database," +
				"by inferring the schema with tools like get-schema and executing arbitrary Cypher queries with read-cypher.",
		},
	)

	neo4jServer.MCPServer = mcpServer

	mcpServer.AddReceivingMiddleware(
		neo4jServer.requestMiddleware,
		neo4jServer.toolsListMiddleware,
		neo4jServer.toolsCallMiddleware,
	)

	return neo4jServer
}

// requestMiddleware logs every incoming request and verifies Neo4j requirements during the
// handshake. Only "initialize" and "server/discover" are checked; other methods pass through
// unchanged. 
func (s *Neo4jMCPServer) requestMiddleware(next mcp.MethodHandler) mcp.MethodHandler {
	return func(ctx context.Context, method string, req mcp.Request) (mcp.Result, error) {
		slog.Info("request started", append(logger.AppendRequestInfo(ctx), "mcp_method", method)...)

		if method != "initialize" && method != "server/discover" {
			return next(ctx, method, req)
		}

		timeout := mcpcontext.GetRequestTimeout(ctx)
		if timeout <= 0 {
			timeout = effectiveRequestTimeout(s.config)
			ctx = mcpcontext.WithRequestTimeout(ctx, timeout)
		}
		ctx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()

		slog.Info("Handshake request: verifying requirements...", "mcp_method", method)
		if err := s.verifyRequirements(ctx); err != nil {
			if isRequestDeadlineExceeded(ctx, err) {
				slog.Warn("request timed out", append(logger.AppendRequestInfo(ctx),
					"mcp_method", method,
					"phase", "initialize",
					"request_timeout_ms", timeout.Milliseconds())...)
				return nil, errors.New(formatRequestTimeoutError(ctx))
			}
			slog.Error("initialize requirements check failed", append(logger.AppendRequestInfo(ctx), "error", err)...)
			return nil, err
		}
		slog.Info("initialize requirements verified", logger.AppendRequestInfo(ctx)...)
		s.emitConnectionInitializedEvent(ctx)

		return next(ctx, method, req)
	}
}

// toolsListMiddleware filters advertised tools depending on the X-Neo4j-MCP-Tools and
// X-Neo4j-MCP-ReadOnly HTTP headers.
func (s *Neo4jMCPServer) toolsListMiddleware(next mcp.MethodHandler) mcp.MethodHandler {
	return func(ctx context.Context, method string, req mcp.Request) (mcp.Result, error) {
		if method != "tools/list" {
			return next(ctx, method, req)
		}

		res, err := next(ctx, method, req)
		if err != nil {
			return res, err
		}

		listResult, ok := res.(*mcp.ListToolsResult)
		if !ok {
			return res, err
		}

		readOnly := mcpcontext.GetReadOnly(ctx)
		requestedTools := mcpcontext.GetTools(ctx)
		// early return when no per-request filters are defined
		if readOnly == nil && requestedTools == nil {
			return listResult, nil
		}

		filteredTools := make([]*mcp.Tool, 0, len(listResult.Tools))
		for _, tool := range listResult.Tools {
			if readOnly != nil && *readOnly && (tool.Annotations == nil || !tool.Annotations.ReadOnlyHint) {
				continue
			}
			if requestedTools != nil && !slices.Contains(*requestedTools, tool.Name) {
				continue
			}
			filteredTools = append(filteredTools, tool)
		}

		if len(filteredTools) != len(listResult.Tools) {
			slog.Debug("tools filtered for request",
				"advertised", len(filteredTools),
				"total", len(listResult.Tools),
			)
		}

		listResult.Tools = filteredTools

		return listResult, nil
	}
}

// toolsCallMiddleware enforces execution-time read-only/tool-list guards, applies the
// per-request timeout, and emits post-call analytics events.
func (s *Neo4jMCPServer) toolsCallMiddleware(next mcp.MethodHandler) mcp.MethodHandler {
	return func(ctx context.Context, method string, req mcp.Request) (mcp.Result, error) {
		if method != "tools/call" {
			return next(ctx, method, req)
		}

		callReq, ok := req.(*mcp.CallToolRequest)
		if !ok {
			return next(ctx, method, req)
		}

		toolName := callReq.Params.Name

		timeout := mcpcontext.GetRequestTimeout(ctx)
		if timeout <= 0 {
			timeout = effectiveRequestTimeout(s.config)
			ctx = mcpcontext.WithRequestTimeout(ctx, timeout)
		}
		ctx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()

		// Guards are checked read-only first, then the configured tools list.
		// When a request violates both, the read-only error takes precedence.
		readOnly := mcpcontext.GetReadOnly(ctx)
		if readOnly != nil && *readOnly {
			tool, ok := s.toolsByName[toolName]
			if !ok {
				// Should be unreachable: the SDK rejects unknown tool names before
				// the middleware runs (see TestHTTPPerRequestToolsExecutionGuardInvalidTool).
				slog.Error("internal error: tool not found", append(logger.AppendRequestInfo(ctx), "tool", toolName)...)
				return nil, fmt.Errorf("internal error: tool %q not found", toolName)
			}

			if tool.Annotations == nil || !tool.Annotations.ReadOnlyHint {
				slog.Warn("tool execution blocked", append(logger.AppendRequestInfo(ctx),
					"tool", toolName, "reason", "read_only")...)
				return tools.NewToolErrorResult(fmt.Sprintf("'%s' is not permitted in read-only mode", toolName)), nil
			}
		}

		requestedTools := mcpcontext.GetTools(ctx)
		if requestedTools != nil && !slices.Contains(*requestedTools, toolName) {
			slog.Warn("tool execution blocked", append(logger.AppendRequestInfo(ctx),
				"tool", toolName, "reason", "not_in_tools_list")...)
			return tools.NewToolErrorResult(fmt.Sprintf("'%s' is not in the list of configured tools", toolName)), nil
		}

		result, err := next(ctx, method, req)
		if isRequestDeadlineExceeded(ctx, err) {
			slog.Warn("request timed out", append(logger.AppendRequestInfo(ctx),
				"mcp_method", "tools/call",
				"tool", toolName,
				"phase", "tool_execution",
				"request_timeout_ms", timeout.Milliseconds())...)
			return tools.NewToolErrorResult(formatRequestTimeoutError(ctx)), nil
		}

		s.handleToolCallComplete(toolName, callReq.Params.Arguments, result)

		return result, err
	}
}

// Start initializes and starts the MCP server
func (s *Neo4jMCPServer) Start() error {
	s.registerTools()
	s.emitServerStartupEvent()
	switch s.config.TransportMode {
	case config.TransportModeHTTP:
		return s.StartHTTPServer()
	case config.TransportModeStdio:
		{
			return s.MCPServer.Run(context.Background(), &mcp.StdioTransport{})
		}
	default:
		return fmt.Errorf("unsupported transport mode: %s", s.config.TransportMode)
	}
}

// healthzHandler handles GET /healthz for infrastructure health checks.
// It requires no authentication and always returns HTTP 200 while the process is alive.
func healthzHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET")
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, err := w.Write([]byte(`{"status":"ok"}`))
	// reasons behind: https://stackoverflow.com/questions/43976140/check-errors-when-calling-http-responsewriter-write
	if err != nil {
		slog.Error("Error writing healthz response", "error", err)
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
func (s *Neo4jMCPServer) verifyRequirements(ctx context.Context) error {
	// Use a timeout to fail fast if the Neo4j instance is unreachable (e.g., TCP connection refused,
	// DNS failure, network failure). Without this, ExecuteReadQuery can block for minutes waiting for
	// the driver's internal connection pool timeout.
	verifyCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	// Perform a dummy query to verify correctness of the connection.
	records, err := s.dbService.ExecuteReadQuery(verifyCtx, "RETURN 1 as first", map[string]any{})

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
	records, err = s.dbService.ExecuteReadQuery(ctx, checkApocMetaSchemaQuery, nil)
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
	if !s.isToolEnabled(ctx, "list-gds-procedures") {
		return nil
	}

	// Call gds.version procedure to determine if GDS is installed
	records, err = s.dbService.ExecuteReadQuery(ctx, "RETURN gds.version() as gdsVersion", nil)
	if err != nil {
		// GDS is optional, so we log a warning and continue, assuming it's not installed.
		slog.Info("Impossible to verify GDS installation.", "error", err)
		return nil
	}
	if len(records) == 1 && len(records[0].Values) == 1 {
		_, ok := records[0].Values[0].(string)
		if ok {
			slog.Info("GDS capability verified")
		}
	}

	return nil
}

// isToolEnabled reports whether a tool is enabled for the current request.
// Per-request HTTP headers take precedence over server configuration.
// When per-request header is present GetTools returns nil. while
// s.config.Tools will always have all the tools available at startup.
func (s *Neo4jMCPServer) isToolEnabled(ctx context.Context, toolName string) bool {
	if tools := mcpcontext.GetTools(ctx); tools != nil {
		return slices.Contains(*tools, toolName)
	}
	return slices.Contains(s.config.Tools, toolName)
}

// emitServerStartupEvent emits the server startup event immediately with available info (no DB query)
func (s *Neo4jMCPServer) emitServerStartupEvent() {
	s.anService.EmitEvent(s.anService.NewStartupEvent(s.config.TransportMode, s.config.HTTPTLSEnabled, s.version))
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
	s.anService.EmitEvent(s.anService.NewConnectionInitializedEvent(connInfo))
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
		versions, ok := versionsRaw.([]any)
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

	slog.Info(
		fmt.Sprintf("Starting Neo4j MCP server version %s in HTTP mode", s.version),
		"version", s.version,
		"listen_url", fmt.Sprintf("%s://%s", protocol, addr),
		"tls", s.config.HTTPTLSEnabled,
	)

	// Create the StreamableHTTPServer - it serves on /mcp path by default
	mcpServerHTTP := mcp.NewStreamableHTTPHandler(
		func(*http.Request) *mcp.Server { return s.MCPServer },
		&mcp.StreamableHTTPOptions{Stateless: true},
	)

	allowedOrigins := parseAllowedOrigins(s.config.HTTPAllowedOrigins)
	writeTimeout := effectiveRequestTimeout(s.config) + httpWriteTimeoutGrace
	shutdownTimeout := writeTimeout + httpShutdownGrace

	// Route /healthz directly (no auth required).
	// All other paths go through the full middleware chain which enforces auth and path validation.
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", healthzHandler)
	mux.Handle("/", s.chainMiddleware(allowedOrigins, mcpServerHTTP))

	// Wrap handler with middleware and create HTTP server
	s.httpServer = &http.Server{
		Addr:    addr,
		Handler: mux,
		// Timeouts optimized for stateless HTTP MCP requests
		ReadTimeout:       serverHTTPReadTimeout,
		WriteTimeout:      writeTimeout,
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
	close(s.HTTPServerReady)

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
		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
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

// handleToolCallComplete is called after every tool call completes
func (s *Neo4jMCPServer) handleToolCallComplete(toolName string, rawArgs json.RawMessage, result mcp.Result) {
	if s.anService == nil || !s.anService.IsEnabled() {
		return
	}

	// Type assert result to *mcp.CallToolResult
	toolResult, ok := result.(*mcp.CallToolResult)
	if !ok {
		return
	}

	// Emit tool event (connection info sent separately in CONNECTION_INITIALIZED event)
	s.anService.EmitEvent(s.anService.NewToolEvent(toolName, !toolResult.IsError))

	// Handle GDS events for cypher tools
	if toolName == "read-cypher" || toolName == "write-cypher" {
		s.emitGDSEventsIfNeeded(rawArgs)
	}
}

// emitGDSEventsIfNeeded checks if the cypher query contains GDS calls and emits appropriate events
func (s *Neo4jMCPServer) emitGDSEventsIfNeeded(rawArgs json.RawMessage) {
	// Arguments arrive as raw JSON at this point in the middleware chain — the typed
	// unmarshal into the tool's Input struct happens later, inside mcp.AddTool's handler.
	var args map[string]any
	if err := json.Unmarshal(rawArgs, &args); err != nil {
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

