// Copyright (c) "Neo4j"
// Neo4j Sweden AB [http://neo4j.com]

package server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"slices"
	"strings"

	"github.com/neo4j/mcp/internal/config"
	"github.com/neo4j/mcp/internal/database"
	"github.com/neo4j/mcp/internal/mcpcontext"
)

const (
	corsMaxAgeSeconds           = "86400" // 24 hours
	maxUnauthenticatedBodyBytes = 4 * 1024
)

var errRequestBodyTooLarge = errors.New("request body too large")

// chainMiddleware chains together all HTTP middleware for this server instance
// last added middleware will be executed first
func (s *Neo4jMCPServer) chainMiddleware(allowedOrigins []string, next http.Handler) http.Handler {
	if s == nil || s.config == nil {
		panic("chainMiddleware: server or config is nil")
	}

	handler := next

	handler = loggingMiddleware()(handler)

	var unauthMethods []string
	if s.config.AllowUnauthenticatedPing {
		unauthMethods = append(unauthMethods, "ping")
	}
	if s.config.AllowUnauthenticatedToolsList {
		unauthMethods = append(unauthMethods, "tools/list")
	}

	if s.uriResolver != nil && s.driverRegistry != nil {
		handler = neo4jDriverMiddleware(s.uriResolver, s.driverRegistry)(handler)
	}

	handler = authMiddleware(s.config.AuthHeaderName, unauthMethods)(handler)
	handler = corsMiddleware(allowedOrigins, s.config.AuthHeaderName)(handler)
	handler = dbNameMiddleware()(handler)
	handler = pathValidationMiddleware()(handler)
	handler = readOnlyMiddleware()(handler)
	handler = toolsMiddleware()(handler)

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

// neo4jDriverMiddleware resolves the Neo4j bolt URI from the header
func neo4jDriverMiddleware(resolver URIResolver, registry database.DriverRegistry) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			uri, err := resolver.Resolve(r)
			if err != nil {
				http.Error(w, "Bad Request: "+err.Error(), http.StatusBadRequest)
				return
			}

			driver, err := registry.GetDriver(uri)
			if err != nil {
				slog.Error("Failed to create Neo4j driver for request", "error", err)
				http.Error(w, "Bad Request: failed to connect to the specified Neo4j instance", http.StatusBadRequest)
				return
			}

			defer func() {
				if closeErr := driver.Close(context.Background()); closeErr != nil {
					slog.Warn("Error closing per-request driver", "error", closeErr)
				}
			}()

			ctx := mcpcontext.WithDriver(r.Context(), driver)
			next.ServeHTTP(w, r.WithContext(ctx))
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
				ctx := mcpcontext.WithBearerToken(r.Context(), token)
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
			ctx := mcpcontext.WithBasicAuth(r.Context(), user, pass)
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

			// Build allowed headers list, always include Content-Type, Authorization, and X-Neo4j-MCP-URI.
			allowedHeaders := []string{"Content-Type", "Authorization", uriHeader}
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

// pathValidationMiddleware validates that requests are only sent to /db/{databaseName}/mcp paths
// and that the HTTP method is allowed. Returns 404 for all other paths to avoid
// hanging connections, and 405 for any method other than POST or OPTIONS since
// the MCP StreamableHTTP Transport spec requires all client messages to be POST
// requests. OPTIONS is permitted so that CORS preflight continues to work.
func pathValidationMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			path := r.URL.Path
			if _, err := parseMCPPath(path); err != nil {
				http.Error(w, "Not Found: This server only handles requests to /db/{databaseName}/mcp", http.StatusNotFound)
				return
			}
			// Only POST and OPTIONS are supported.
			if r.Method != http.MethodPost && r.Method != http.MethodOptions {
				w.Header().Set("Allow", "POST, OPTIONS")
				http.Error(w, "Method Not Allowed: only POST and OPTIONS is supported on /db/{databaseName}/mcp", http.StatusMethodNotAllowed)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// readOnlyMiddleware reads the "X-Neo4j-MCP-ReadOnly" request header and stores
// the resulting boolean in the request context. Accepted values are "true" and
// "false" (case-insensitive). Any other non-empty value yields a 400 Bad Request.
// When the header is absent it does not set the ReadOnly in the context.
func readOnlyMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// r.Header.Values is used to distinguish when the header is not set
			vals := r.Header.Values("X-Neo4j-MCP-ReadOnly")

			if len(vals) > 1 {
				http.Error(w, "Bad Request: duplicate X-Neo4j-MCP-ReadOnly header found", http.StatusBadRequest)
				return
			} else if len(vals) == 1 {
				switch strings.ToLower(vals[0]) {
				case "false":
					ctx := mcpcontext.WithReadOnly(r.Context(), false)
					next.ServeHTTP(w, r.WithContext(ctx))
					return
				case "true":
					ctx := mcpcontext.WithReadOnly(r.Context(), true)
					next.ServeHTTP(w, r.WithContext(ctx))
					return
				default:
					http.Error(w, `Bad Request: "X-Neo4j-MCP-ReadOnly" must be "true" or "false"`, http.StatusBadRequest)
					return
				}

			}
			next.ServeHTTP(w, r)
		})
	}
}

// toolsMiddleware reads the "X-Neo4j-MCP-Tools" request header and stores
// the resulting values in the request context. Accepted values are the tools
// supported by the Neo4j MCP Server such as: "read-cypher", "get-schema".
// Any other non-empty value yields a 400 Bad Request.
// When the header is absent it does not set the Tools in the context.
func toolsMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// r.Header.Values is used to distinguish when the header is not set
			vals := r.Header.Values("X-Neo4j-MCP-Tools")
			if len(vals) > 1 {
				http.Error(w, "Bad Request: duplicate X-Neo4j-MCP-Tools header found", http.StatusBadRequest)
				return
			} else if len(vals) == 1 {
				tools := parseCommaSeparatedString(vals[0])
				if len(tools) == 0 {
					http.Error(w, fmt.Sprintf("tool %q is invalid. Available tools are: %s", vals[0], strings.Join(config.AvailableTools, ", ")), http.StatusBadRequest)
					return
				}

				for _, toolName := range tools {
					if !slices.Contains(config.AvailableTools, toolName) {
						http.Error(w, fmt.Sprintf("tool %q is invalid. Available tools are: %s", toolName, strings.Join(config.AvailableTools, ", ")), http.StatusBadRequest)
						return
					}
				}
				ctx := mcpcontext.WithTools(r.Context(), tools)
				next.ServeHTTP(w, r.WithContext(ctx))
				return

			}
			next.ServeHTTP(w, r)
		})
	}
}

// dbNameMiddleware extracts and validates the database name from the URL path,
// storing it in the request context.
func dbNameMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			database, err := parseMCPPath(r.URL.Path)
			if err != nil {
				http.Error(w, "Bad Request: Invalid path", http.StatusBadRequest)
				return
			}

			if !isValidDatabaseName(database) {
				http.Error(w, "Bad Request: Invalid database name", http.StatusBadRequest)
				return
			}

			ctx := mcpcontext.WithDatabaseName(r.Context(), database)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// parseCommaSeparatedString parses a comma-separated string into a slice of strings.
// Ensures that whitespace, trailing and leading commas are ignored.
func parseCommaSeparatedString(value string) []string {
	parts := strings.Split(value, ",")
	n := 0
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			parts[n] = p
			n++
		}
	}
	return parts[:n]
}
