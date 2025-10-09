package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/neo4j/mcp/internal/config"
)

func TestJWTAuthMiddleware_NoAuth0Config(t *testing.T) {
	// Create server without Auth0 configuration
	cfg := &config.Config{
		Auth0Domain:   "",
		Auth0Audience: "",
	}

	server := &Neo4jMCPServer{
		config: cfg,
	}

	// Create a test handler that will be called if middleware passes
	nextCalled := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusOK)
	})

	// Wrap with JWT middleware
	handler := server.jwtAuthMiddleware(next)

	// Create test request without Authorization header
	req := httptest.NewRequest(http.MethodGet, "/mcp", nil)
	rec := httptest.NewRecorder()

	// Execute request
	handler.ServeHTTP(rec, req)

	// Should pass through when Auth0 is not configured
	if !nextCalled {
		t.Error("Expected next handler to be called when Auth0 is not configured")
	}

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}
}

func TestJWTAuthMiddleware_WithAuth0Config_NoToken(t *testing.T) {
	// Create server with Auth0 configuration
	cfg := &config.Config{
		Auth0Domain:   "test.auth0.com",
		Auth0Audience: "https://test-api",
	}

	server := &Neo4jMCPServer{
		config: cfg,
	}

	// Create a test handler
	nextCalled := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusOK)
	})

	// Wrap with JWT middleware
	handler := server.jwtAuthMiddleware(next)

	// Create test request without Authorization header
	req := httptest.NewRequest(http.MethodGet, "/mcp", nil)
	rec := httptest.NewRecorder()

	// Execute request
	handler.ServeHTTP(rec, req)

	// Should reject when token is missing
	if nextCalled {
		t.Error("Expected next handler NOT to be called when token is missing")
	}

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", rec.Code)
	}

	// Verify WWW-Authenticate header is present per RFC9728 Section 5.1
	wwwAuth := rec.Header().Get("WWW-Authenticate")
	if wwwAuth == "" {
		t.Error("Expected WWW-Authenticate header to be present in 401 response per RFC9728 Section 5.1")
	}

	// Verify header contains required components
	expectedRealm := `realm="https://test.auth0.com/.well-known/oauth-authorization-server"`
	if !strings.Contains(wwwAuth, expectedRealm) {
		t.Errorf("Expected WWW-Authenticate header to contain %s, got: %s", expectedRealm, wwwAuth)
	}

	if !strings.Contains(wwwAuth, `error="invalid_request"`) {
		t.Errorf("Expected WWW-Authenticate header to contain error code, got: %s", wwwAuth)
	}

	if !strings.Contains(wwwAuth, `error_description=`) {
		t.Errorf("Expected WWW-Authenticate header to contain error description, got: %s", wwwAuth)
	}
}

func TestJWTAuthMiddleware_WithAuth0Config_InvalidToken(t *testing.T) {
	// Create server with Auth0 configuration
	cfg := &config.Config{
		Auth0Domain:   "test.auth0.com",
		Auth0Audience: "https://test-api",
	}

	server := &Neo4jMCPServer{
		config: cfg,
	}

	// Create a test handler
	nextCalled := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusOK)
	})

	// Wrap with JWT middleware
	handler := server.jwtAuthMiddleware(next)

	// Create test request with invalid Authorization header
	req := httptest.NewRequest(http.MethodGet, "/mcp", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	rec := httptest.NewRecorder()

	// Execute request
	handler.ServeHTTP(rec, req)

	// Should reject when token is invalid
	if nextCalled {
		t.Error("Expected next handler NOT to be called when token is invalid")
	}

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", rec.Code)
	}

	// Verify WWW-Authenticate header is present per RFC9728 Section 5.1
	wwwAuth := rec.Header().Get("WWW-Authenticate")
	if wwwAuth == "" {
		t.Error("Expected WWW-Authenticate header to be present in 401 response per RFC9728 Section 5.1")
	}

	// Verify header contains the resource server metadata URL
	expectedRealm := `realm="https://test.auth0.com/.well-known/oauth-authorization-server"`
	if !strings.Contains(wwwAuth, expectedRealm) {
		t.Errorf("Expected WWW-Authenticate header to contain %s, got: %s", expectedRealm, wwwAuth)
	}

	// For invalid token, should include error="invalid_token"
	if !strings.Contains(wwwAuth, `error="invalid_token"`) {
		t.Errorf("Expected WWW-Authenticate header to contain error=\"invalid_token\", got: %s", wwwAuth)
	}
}

func TestJWTAuthMiddleware_WithAuth0Config_InvalidAuthHeaderFormat(t *testing.T) {
	// Create server with Auth0 configuration
	cfg := &config.Config{
		Auth0Domain:   "test.auth0.com",
		Auth0Audience: "https://test-api",
	}

	server := &Neo4jMCPServer{
		config: cfg,
	}

	// Create a test handler
	nextCalled := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusOK)
	})

	// Wrap with JWT middleware
	handler := server.jwtAuthMiddleware(next)

	// Create test request with invalid Authorization header format (not Bearer)
	req := httptest.NewRequest(http.MethodGet, "/mcp", nil)
	req.Header.Set("Authorization", "Basic dXNlcjpwYXNz")
	rec := httptest.NewRecorder()

	// Execute request
	handler.ServeHTTP(rec, req)

	// Should reject when token format is invalid
	if nextCalled {
		t.Error("Expected next handler NOT to be called when auth header format is invalid")
	}

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", rec.Code)
	}

	// Verify WWW-Authenticate header is present per RFC9728 Section 5.1
	wwwAuth := rec.Header().Get("WWW-Authenticate")
	if wwwAuth == "" {
		t.Error("Expected WWW-Authenticate header to be present in 401 response per RFC9728 Section 5.1")
	}

	// Verify header contains the resource server metadata URL
	expectedRealm := `realm="https://test.auth0.com/.well-known/oauth-authorization-server"`
	if !strings.Contains(wwwAuth, expectedRealm) {
		t.Errorf("Expected WWW-Authenticate header to contain %s, got: %s", expectedRealm, wwwAuth)
	}

	// For invalid request format, should include error="invalid_request"
	if !strings.Contains(wwwAuth, `error="invalid_request"`) {
		t.Errorf("Expected WWW-Authenticate header to contain error=\"invalid_request\", got: %s", wwwAuth)
	}
}
