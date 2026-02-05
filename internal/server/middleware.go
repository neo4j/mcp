package server

import (
	"log/slog"
	"net/http"
	"slices"
	"strings"

	"github.com/neo4j/mcp/internal/auth"
)

const (
	corsMaxAgeSeconds = "86400" // 24 hours
)

// chainMiddleware chains together all HTTP middleware for this server instance
func (s *Neo4jMCPServer) chainMiddleware(allowedOrigins []string, next http.Handler) http.Handler {
	// Chain middleware in reverse order (last added = first to execute)
	// Execution order: PathValidator -> CORS -> Auth (Bearer/Basic) -> Logging -> Handler

	// Start with the actual handler
	handler := next

	// Add logging middleware
	handler = loggingMiddleware()(handler)

	// Add auth middleware (supports both Bearer and Basic authentication)
	handler = authMiddleware()(handler)

	// Add CORS middleware (if configured)
	handler = corsMiddleware(allowedOrigins)(handler)

	// Add path validation middleware last (executes first - reject non-/mcp paths quickly)
	handler = pathValidationMiddleware()(handler)

	return handler
}

// authMiddleware enforces HTTP authentication (Bearer token or Basic Auth) for all requests in HTTP mode.
// Tries Bearer token first (from Authorization: Bearer header), then falls back to Basic Auth.
// Credentials are extracted and stored in the request context for tools to create
// per-request Neo4j driver connections, enabling multi-tenant scenarios.
// Returns 401 Unauthorized if credentials are missing or malformed.
func authMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")

			// Try bearer token first
			if strings.HasPrefix(authHeader, "Bearer ") {
				token := strings.TrimPrefix(authHeader, "Bearer ")
				token = strings.TrimSpace(token)

				if token == "" {
					w.Header().Set("WWW-Authenticate", `Bearer realm="Neo4j MCP Server"`)
					http.Error(w, "Unauthorized: Bearer token is empty", http.StatusUnauthorized)
					return
				}

				// Bearer token provided - store in context
				ctx := auth.WithBearerToken(r.Context(), token)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			// Fall back to basic auth
			user, pass, ok := r.BasicAuth()
			if !ok {
				// No credentials provided - reject request
				w.Header().Add("WWW-Authenticate", `Basic realm="Neo4j MCP Server"`)
				w.Header().Add("WWW-Authenticate", `Bearer realm="Neo4j MCP Server"`)
				http.Error(w, "Unauthorized: Basic or Bearer authentication required", http.StatusUnauthorized)
				return
			}

			// Validate credentials are not empty (consistent with bearer token validation)
			if user == "" || pass == "" {
				w.Header().Set("WWW-Authenticate", `Basic realm="Neo4j MCP Server"`)
				http.Error(w, "Unauthorized: Username and password cannot be empty", http.StatusUnauthorized)
				return
			}

			// Basic auth credentials provided - store in context
			ctx := auth.WithBasicAuth(r.Context(), user, pass)
			next.ServeHTTP(w, r.WithContext(ctx))
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
			if r.URL.Path != "/mcp" && r.URL.Path != "/mcp/" {
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
