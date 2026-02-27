// Copyright (c) "Neo4j"
// Neo4j Sweden AB [http://neo4j.com]

package server

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
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
	// Execution order: PathValidator -> CORS -> Auth (Bearer/Basic) -> Logging -> Handler

	// Start with the actual handler
	handler := next

	// Add logging middleware
	handler = loggingMiddleware()(handler)

	handler = authMiddleware(s.config.AuthHeaderName, s.config.AllowUnauthenticatedPing)(handler)

	// Add CORS middleware (if configured) - includes Mcp-Session-Id in allowed headers
	handler = corsMiddleware(allowedOrigins, s.config.AuthHeaderName)(handler)

	// Add path validation middleware last (executes first - reject non-/mcp paths quickly)
	handler = pathValidationMiddleware()(handler)

	return handler
}

// authMiddleware enforces HTTP authentication (Bearer token or Basic Auth) for all requests in HTTP mode.
// Tries Bearer token first (from Authorization: Bearer header), then falls back to Basic Auth.
// Credentials are extracted and stored in the request context for tools to create
// per-request Neo4j driver connections, enabling multi-tenant scenarios.
// Returns 401 Unauthorized if credentials are missing or malformed.
func authMiddleware(headerName string, allowUnauthenticatedPing bool) func(http.Handler) http.Handler {
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
				if allowUnauthenticatedPing {
					// Wrap the body; this will cause reads past the limit to fail. Only apply
					// when the feature is enabled to avoid unintended side effects on other endpoints.
					r.Body = http.MaxBytesReader(w, r.Body, maxUnauthenticatedBodyBytes)

					ok, err := isUnauthenticatedPingRequest(r)
					if err != nil {
						// If the body was too large, return 413 Payload Too Large
						if errors.Is(err, errRequestBodyTooLarge) {
							http.Error(w, err.Error(), http.StatusRequestEntityTooLarge)
							return
						}
						// For other read errors or JSON errors, fall through and require auth
					}

					if ok {
						next.ServeHTTP(w, r)
						return
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

// pathValidationMiddleware validates that requests are only sent to /mcp path
// and that the HTTP method is allowed. Returns 404 for all other paths to avoid
// hanging connections, and 405 for GET requests since this server does not offer
// an SSE stream (as required by the MCP spec).
func pathValidationMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Only /mcp path is valid for this MCP server
			if r.URL.Path != "/mcp" && r.URL.Path != "/mcp/" {
				http.Error(w, "Not Found: This server only handles requests to /mcp", http.StatusNotFound)
				return
			}
			// GET is not supported: per the MCP spec the server MUST return either
			// text/event-stream or 405. This server does not offer an SSE stream.
			if r.Method == http.MethodGet {
				w.Header().Set("Allow", "POST, OPTIONS")
				http.Error(w, "Method Not Allowed: GET is not supported on /mcp", http.StatusMethodNotAllowed)
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

func isUnauthenticatedPingRequest(r *http.Request) (bool, error) {
	if r.Method != http.MethodPost {
		return false, nil
	}
	if r.ContentLength >= 0 && r.ContentLength > maxUnauthenticatedBodyBytes {
		return false, errRequestBodyTooLarge
	}

	buf, err := io.ReadAll(r.Body)
	// Close the original body to free resources.
	if rc := r.Body; rc != nil {
		_ = rc.Close()
	}

	if err != nil {
		// replace body with an empty reader to avoid further reads.
		r.Body = io.NopCloser(bytes.NewReader(nil))

		// If MaxBytesReader triggered, it typically returns an error containing
		// "request body too large". Map that to a sentinel error so middleware can
		// respond with 413.
		if strings.Contains(err.Error(), "request body too large") {
			return false, errRequestBodyTooLarge
		}

		return false, err
	}

	// Restore the read bytes so downstream handlers can read the body as usual
	r.Body = io.NopCloser(bytes.NewReader(buf))

	var probe struct {
		Method string `json:"method"`
	}
	if e := json.Unmarshal(buf, &probe); e != nil {
		return false, e
	}

	return probe.Method == "ping", nil
}
