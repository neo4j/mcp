// Copyright (c) "Neo4j"
// Neo4j Sweden AB [http://neo4j.com]

package server

import (
	"errors"
	"log/slog"
	"net/http"
	"slices"
	"strings"

	"github.com/neo4j/mcp/internal/auth"
)

const (
	corsMaxAgeSeconds           = "86400" // 24 hours
	maxUnauthenticatedBodyBytes = 4 * 1024
)

var errRequestBodyTooLarge = errors.New("request body too large")

// chainMiddleware chains together all HTTP middleware for this server instance
func (s *Neo4jMCPServer) chainMiddleware(allowedOrigins []string, next http.Handler) http.Handler {
	if s == nil || s.config == nil {
		panic("chainMiddleware: server or config is nil")
	}

	// Chain middleware in reverse order (last added = first to execute)
	// Middleware execution order: DB name extractor -> Path validation -> CORS -> Auth (Bearer/Basic) -> Logging -> Handler

	handler := next

	handler = loggingMiddleware()(handler)

	var unauthMethods []string
	if s.config.AllowUnauthenticatedPing {
		unauthMethods = append(unauthMethods, "ping")
	}
	if s.config.AllowUnauthenticatedToolsList {
		unauthMethods = append(unauthMethods, "tools/list")
	}

	handler = authMiddleware(s.config.AuthHeaderName, unauthMethods)(handler)
	handler = corsMiddleware(allowedOrigins, s.config.AuthHeaderName)(handler)
	handler = pathValidationMiddleware()(handler)
	handler = dbNameMiddleware()(handler)

	return handler
}

// loggingMiddleware logs HTTP requests for debugging
func loggingMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			slog.Debug("HTTP Request", // #nosec G706 -- logging HTTP request metadata, no user input in format string
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

// authMiddleware enforces HTTP authentication (Bearer token or Basic Auth) for all requests in HTTP mode.
// Tries Bearer token first (from Authorization: Bearer header), then falls back to Basic Auth.
// Credentials are extracted and stored in the request context for tools to create
// per-request Neo4j driver connections, enabling multi-tenant scenarios.
// unauthenticatedMethods is an optional list of JSON-RPC method names (e.g. "ping", "tools/list")
// that are permitted without credentials.
// Returns 401 Unauthorized if credentials are missing or malformed.
func authMiddleware(headerName string, unauthenticatedMethods []string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			if !strings.EqualFold(headerName, "Authorization") {
				val := r.Header.Get(headerName)
				if val != "" {
					r.Header.Set("Authorization", val)
				}
			}

			authHeader := r.Header.Get("Authorization")

			// Try the bearer token first
			if token, found := strings.CutPrefix(authHeader, "Bearer "); found {
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
				if len(unauthenticatedMethods) > 0 {
					// Wrap the body once to enforce a size limit for unauthenticated probes.
					r.Body = http.MaxBytesReader(w, r.Body, maxUnauthenticatedBodyBytes)

					for _, method := range unauthenticatedMethods {
						ok, err := isUnauthenticatedMethodRequest(r, method)
						if err != nil {
							if errors.Is(err, errRequestBodyTooLarge) {
								http.Error(w, err.Error(), http.StatusRequestEntityTooLarge)
								return
							}
							// For other read errors or JSON errors, fall through and require auth
							continue
						}
						if ok {
							next.ServeHTTP(w, r)
							return
						}
					}
				}

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
func corsMiddleware(allowedOrigins []string, authHeaderName string) func(http.Handler) http.Handler {
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

			// Build allowed headers list, always include Content-Type and Authorization.
			allowedHeaders := []string{"Content-Type", "Authorization"}
			// If a custom auth header is configured, and it's not the default, include it
			if authHeaderName != "" && !strings.EqualFold(authHeaderName, "Authorization") {
				allowedHeaders = append(allowedHeaders, authHeaderName)
			}

			w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", strings.Join(allowedHeaders, ", "))
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

// pathValidationMiddleware validates that requests are only sent to /mcp or /db/{databaseName}/mcp paths
// and that the HTTP method is allowed. Returns 404 for all other paths to avoid
// hanging connections, and 405 for any method other than POST or OPTIONS since
// the MCP StreamableHTTP Transport spec requires all client messages to be POST
// requests. OPTIONS is permitted so that CORS preflight continues to work.
func pathValidationMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			path := r.URL.Path
			// Allow /mcp, /mcp/, and /db/{databaseName}/mcp paths
			if _, ok := parseMCPPath(path); !ok {
				http.Error(w, "Not Found: This server only handles requests to /mcp or /db/{databaseName}/mcp", http.StatusNotFound)
				return
			}
			// Only POST and OPTIONS are supported.
			if r.Method != http.MethodPost && r.Method != http.MethodOptions {
				w.Header().Set("Allow", "POST, OPTIONS")
				http.Error(w, "Method Not Allowed: only POST is supported on /mcp", http.StatusMethodNotAllowed)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// dbNameMiddleware extracts the database name from the URL path and stores it in the request context.
// Only processes /db/{databaseName}/mcp paths; passes through /mcp requests without modification.
func dbNameMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			database, _ := parseMCPPath(r.URL.Path)
			if database == "" {
				next.ServeHTTP(w, r)
				return
			}

			if !isValidDatabaseName(database) {
				http.Error(w, "Bad Request: Invalid database name", http.StatusBadRequest)
				return
			}

			ctx := r.Context()
			ctx = auth.WithDatabaseName(ctx, database)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
