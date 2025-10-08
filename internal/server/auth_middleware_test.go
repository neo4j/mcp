package server

import (
	"net/http"
	"net/http/httptest"
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
}
