package server

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"slices"

	"github.com/neo4j/mcp/internal/auth"
	"github.com/neo4j/mcp/internal/config"
)

// mcpRequest represents the minimal JSON-RPC structure needed to extract the method.
// MCP uses JSON-RPC 2.0 for all protocol messages.
type mcpRequest struct {
	Method string `json:"method"`
}

// mcpMethodsRequiringAuth lists MCP methods that require authentication.
// These are methods that access the Neo4j database and need valid credentials.
// All other methods (protocol handshake, capability exchange) are allowed without auth.
var mcpMethodsRequiringAuth = []string{
	"tools/call", // Tool execution - requires database credentials
}

// isAuthRequiredForMethod determines if an MCP method requires authentication.
// Returns true for methods that access the Neo4j database (tools/call).
// Returns false for protocol handshake methods (initialize, tools/list, etc.)
// that only describe server capabilities without accessing the database.
func isAuthRequiredForMethod(method string) bool {
	for _, m := range mcpMethodsRequiringAuth {
		if method == m {
			return true
		}
	}
	return false
}

// extractMCPMethod reads the request body, extracts the JSON-RPC method,
// and restores the body for subsequent handlers.
// Returns the method name and any error encountered.
func extractMCPMethod(r *http.Request) (string, error) {
	if r.Body == nil {
		return "", nil
	}

	// Read the body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return "", err
	}

	// Restore the body for subsequent handlers
	r.Body = io.NopCloser(bytes.NewReader(body))

	// Handle empty body (e.g., GET requests)
	if len(body) == 0 {
		return "", nil
	}

	// Parse the JSON-RPC request to extract the method
	var req mcpRequest
	if err := json.Unmarshal(body, &req); err != nil {
		// If we can't parse it, let it through without auth
		// The MCP handler will deal with malformed requests
		slog.Debug("Could not parse request as JSON-RPC", "error", err)
		return "", nil
	}

	return req.Method, nil
}

const (
	corsMaxAgeSeconds = "86400" // 24 hours
)

// chainMiddleware chains together all HTTP middleware
func chainMiddleware(cfg *config.Config, allowedOrigins []string, next http.Handler) http.Handler {
	// Chain middleware in reverse order (last added = first to execute)
	// Execution order: PathValidator -> CORS -> BasicAuth -> Logging -> Handler

	// Start with the actual handler
	handler := next

	// Add logging middleware
	handler = loggingMiddleware()(handler)

	// Add basic auth middleware - uses request header credentials or falls back to config
	handler = basicAuthMiddleware(cfg)(handler)

	// Add CORS middleware (if configured)
	handler = corsMiddleware(allowedOrigins)(handler)

	// Add path validation middleware last (executes first - reject non-/mcp paths quickly)
	handler = pathValidationMiddleware()(handler)

	return handler
}

// basicAuthMiddleware enforces HTTP Basic Authentication for MCP methods that
// require database access (tools/call). Protocol handshake methods (initialize,
// tools/list, etc.) are allowed without authentication to enable platform
// health checks and capability discovery.
//
// Credentials can come from two sources (in priority order):
// 1. Request Authorization header (Basic Auth)
// 2. Environment variables (NEO4J_USERNAME, NEO4J_PASSWORD) as fallback
//
// When credentials are provided, they are stored in the request context for
// tools to create per-request Neo4j driver connections.
//
// Returns 401 Unauthorized only for methods that require auth when no credentials
// are available from either source.
func basicAuthMiddleware(cfg *config.Config) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract the MCP method from the request body
			method, err := extractMCPMethod(r)
			if err != nil {
				slog.Warn("Failed to extract MCP method from request", "error", err)
				// On error, require auth as a safe default
				w.Header().Set("WWW-Authenticate", `Basic realm="Neo4j MCP Server"`)
				http.Error(w, "Unauthorized: Basic authentication required", http.StatusUnauthorized)
				return
			}

			// Check if this method requires authentication
			authRequired := isAuthRequiredForMethod(method)

			// Get credentials from Authorization header (priority) or fall back to config
			user, pass, hasCredentials := r.BasicAuth()
			if !hasCredentials && cfg.Username != "" && cfg.Password != "" {
				// Fall back to environment variable credentials
				user = cfg.Username
				pass = cfg.Password
				hasCredentials = true
				slog.Debug("Using environment variable credentials as fallback")
			}

			if authRequired && !hasCredentials {
				// Method requires auth but no credentials available
				slog.Debug("Authentication required for method", "method", method)
				w.Header().Set("WWW-Authenticate", `Basic realm="Neo4j MCP Server"`)
				http.Error(w, "Unauthorized: Basic authentication required for database operations", http.StatusUnauthorized)
				return
			}

			// If credentials are available, store them in context (even for non-auth-required methods)
			// This allows tools to use credentials if they happen to be provided
			if hasCredentials {
				ctx := auth.WithBasicAuth(r.Context(), user, pass)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			// No credentials and not required - allow request
			slog.Debug("Allowing unauthenticated request for protocol method", "method", method)
			next.ServeHTTP(w, r)
		})
	}
}

// corsMiddleware implements CORS (Cross-Origin Resource Sharing)
// If allowedOrigins is empty, CORS is disabled
// If allowedOrigins is "*", all origins are allowed
// Otherwise, allowedOrigins should be a comma-separated list of allowed origins
func corsMiddleware(allowedOrigins []string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip CORS if not configured
			if len(allowedOrigins) == 0 {
				next.ServeHTTP(w, r)
				return
			}

			origin := r.Header.Get("Origin")

			// Handle wildcard case
			if slices.Contains(allowedOrigins, "*") {
				w.Header().Set("Access-Control-Allow-Origin", "*")
			} else if origin != "" && slices.Contains(allowedOrigins, origin) {
				// Check if the request origin is allowed
				w.Header().Set("Access-Control-Allow-Origin", origin)
			}

			// Set other CORS headers
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			w.Header().Set("Access-Control-Max-Age", corsMaxAgeSeconds)

			// Handle preflight requests
			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// pathValidationMiddleware validates that requests are only sent to /mcp path
// Returns 404 for all other paths to avoid hanging connections
func pathValidationMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Only /mcp path is valid for this MCP server
			if r.URL.Path != "/mcp" {
				http.Error(w, "Not Found: This server only handles requests to /mcp", http.StatusNotFound)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// loggingMiddleware logs HTTP requests for debugging
func loggingMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			slog.Debug("HTTP Request",
				"method", r.Method,
				"url", r.URL.Path,
				"remote_addr", r.RemoteAddr,
				"user_agent", r.UserAgent(),
				"content_length", r.ContentLength,
				"host", r.Host,
				"query", r.URL.RawQuery,
			)

			// Call the next handler
			next.ServeHTTP(w, r)
		})
	}
}
