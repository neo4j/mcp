package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/mark3labs/mcp-go/server"
	"github.com/neo4j/mcp/internal/analytics"
	analytics_mocks "github.com/neo4j/mcp/internal/analytics/mocks"
	"github.com/neo4j/mcp/internal/auth"
	"github.com/neo4j/mcp/internal/config"
	db_mocks "github.com/neo4j/mcp/internal/database/mocks"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"go.uber.org/mock/gomock"
)

// mockHandler is a simple handler that returns 200 OK
func mockHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})
}

// mockNeo4jMCPServer creates a mock Neo4jMCPServer for testing
func mockNeo4jMCPServer(t *testing.T) *Neo4jMCPServer {
	t.Helper()
	ctrl := gomock.NewController(t)

	cfg := &config.Config{
		URI:           "bolt://localhost:7687",
		Username:      "neo4j",
		Password:      "password",
		Database:      "neo4j",
		TransportMode: config.TransportModeHTTP,
		Telemetry:     false, // Disable telemetry in tests
	}

	mockDBService := db_mocks.NewMockService(ctrl)
	mockAnalyticsService := analytics_mocks.NewMockService(ctrl)

	mcpServer := server.NewMCPServer("test-server", "1.0.0")

	return &Neo4jMCPServer{
		MCPServer:    mcpServer,
		config:       cfg,
		dbService:    mockDBService,
		anService:    mockAnalyticsService,
		version:      "1.0.0",
		gdsInstalled: false,
	}
}

// authCheckHandler verifies if credentials are in context
func authCheckHandler(t *testing.T, expectAuth bool, expectedUser, expectedPass string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, pass, ok := auth.GetBasicAuthCredentials(r.Context())
		if expectAuth {
			if !ok {
				t.Error("Expected auth credentials in context, but none found")
			}
			if user != expectedUser {
				t.Errorf("Expected user %q, got %q", expectedUser, user)
			}
			if pass != expectedPass {
				t.Errorf("Expected pass %q, got %q", expectedPass, pass)
			}
		} else if ok {
			t.Error("Expected no auth credentials in context, but found some")
		}
		w.WriteHeader(http.StatusOK)
	})
}

func TestBasicAuthMiddleware_WithValidCredentials(t *testing.T) {
	handler := basicAuthMiddleware()(authCheckHandler(t, true, "testuser", "testpass"))

	req := httptest.NewRequest("GET", "/", nil)
	req.SetBasicAuth("testuser", "testpass")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}
}

func TestBasicAuthMiddleware_WithoutCredentials(t *testing.T) {
	handler := basicAuthMiddleware()(mockHandler())

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	// Should return 401 when no credentials provided
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", rec.Code)
	}

	// Should have WWW-Authenticate header
	if rec.Header().Get("WWW-Authenticate") == "" {
		t.Error("Expected WWW-Authenticate header to be set")
	}
}

func TestBasicAuthMiddleware_WithEmptyCredentials(t *testing.T) {
	handler := basicAuthMiddleware()(authCheckHandler(t, true, "", ""))

	req := httptest.NewRequest("GET", "/", nil)
	req.SetBasicAuth("", "")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}
}

func TestCORSMiddleware_NoConfiguration(t *testing.T) {
	handler := corsMiddleware([]string{})(mockHandler())

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Origin", "http://example.com")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	// No CORS headers should be set
	if rec.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Error("Expected no CORS headers when CORS is not configured")
	}
}

func TestCORSMiddleware_WildcardOrigin(t *testing.T) {
	handler := corsMiddleware([]string{"*"})(mockHandler())

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Origin", "http://example.com")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	if rec.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Errorf("Expected Access-Control-Allow-Origin: *, got %q", rec.Header().Get("Access-Control-Allow-Origin"))
	}
}

func TestCORSMiddleware_SpecificOriginMatching(t *testing.T) {
	allowedOrigins := []string{"http://example.com", "http://localhost:3000"}
	handler := corsMiddleware(allowedOrigins)(mockHandler())

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Origin", "http://example.com")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	if rec.Header().Get("Access-Control-Allow-Origin") != "http://example.com" {
		t.Errorf("Expected Access-Control-Allow-Origin: http://example.com, got %q", rec.Header().Get("Access-Control-Allow-Origin"))
	}
}

func TestCORSMiddleware_SpecificOriginNotMatching(t *testing.T) {
	allowedOrigins := []string{"http://example.com"}
	handler := corsMiddleware(allowedOrigins)(mockHandler())

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Origin", "http://evil.com")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	// Origin should not be set for non-matching origins
	if rec.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Error("Expected no Access-Control-Allow-Origin header for non-matching origin")
	}

	// But other CORS headers should still be present
	if rec.Header().Get("Access-Control-Allow-Methods") == "" {
		t.Error("Expected Access-Control-Allow-Methods header to be set")
	}
}

func TestCORSMiddleware_MultipleOrigins(t *testing.T) {
	allowedOrigins := []string{"http://example.com", "http://localhost:3000", "http://test.com"}
	handler := corsMiddleware(allowedOrigins)(mockHandler())

	testCases := []struct {
		origin   string
		expected string
	}{
		{"http://example.com", "http://example.com"},
		{"http://localhost:3000", "http://localhost:3000"},
		{"http://test.com", "http://test.com"},
		{"http://notallowed.com", ""},
	}

	for _, tc := range testCases {
		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set("Origin", tc.origin)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200 for origin %s, got %d", tc.origin, rec.Code)
		}

		actual := rec.Header().Get("Access-Control-Allow-Origin")
		if actual != tc.expected {
			t.Errorf("For origin %s, expected Access-Control-Allow-Origin: %q, got %q", tc.origin, tc.expected, actual)
		}
	}
}

func TestCORSMiddleware_PreflightRequest(t *testing.T) {
	allowedOrigins := []string{"http://example.com"}
	handler := corsMiddleware(allowedOrigins)(mockHandler())

	req := httptest.NewRequest("OPTIONS", "/", nil)
	req.Header.Set("Origin", "http://example.com")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Errorf("Expected status 204 for OPTIONS request, got %d", rec.Code)
	}

	if rec.Header().Get("Access-Control-Allow-Origin") != "http://example.com" {
		t.Errorf("Expected Access-Control-Allow-Origin: http://example.com, got %q", rec.Header().Get("Access-Control-Allow-Origin"))
	}

	if rec.Header().Get("Access-Control-Allow-Methods") == "" {
		t.Error("Expected Access-Control-Allow-Methods header to be set")
	}

	if rec.Header().Get("Access-Control-Allow-Headers") == "" {
		t.Error("Expected Access-Control-Allow-Headers header to be set")
	}

	if rec.Header().Get("Access-Control-Max-Age") != corsMaxAgeSeconds {
		t.Errorf("Expected Access-Control-Max-Age: %s, got %q", corsMaxAgeSeconds, rec.Header().Get("Access-Control-Max-Age"))
	}
}

func TestCORSMiddleware_MissingOriginHeader(t *testing.T) {
	allowedOrigins := []string{"http://example.com"}
	handler := corsMiddleware(allowedOrigins)(mockHandler())

	req := httptest.NewRequest("GET", "/", nil)
	// No Origin header set
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	// No origin header should be set when request has no Origin
	if rec.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Error("Expected no Access-Control-Allow-Origin header when request has no Origin")
	}
}

func TestLoggingMiddleware(t *testing.T) {
	handler := loggingMiddleware()(mockHandler())

	req := httptest.NewRequest("GET", "/test?foo=bar", nil)
	req.Header.Set("User-Agent", "test-agent")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	// Logging middleware should not modify the response
	if rec.Body.String() != "OK" {
		t.Errorf("Expected body 'OK', got %q", rec.Body.String())
	}
}

func TestAddMiddleware_FullChain(t *testing.T) {
	allowedOrigins := []string{"http://example.com"}
	mockServer := mockNeo4jMCPServer(t)
	handler := mockServer.chainMiddleware(allowedOrigins, authCheckHandler(t, true, "user", "pass"))

	req := httptest.NewRequest("GET", "/mcp", nil)
	req.Header.Set("Origin", "http://example.com")
	req.SetBasicAuth("user", "pass")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	// Verify CORS headers are set
	if rec.Header().Get("Access-Control-Allow-Origin") != "http://example.com" {
		t.Errorf("Expected CORS header to be set")
	}
}

func TestAddMiddleware_FullChain_NoAuth(t *testing.T) {
	allowedOrigins := []string{"http://example.com"}
	mockServer := mockNeo4jMCPServer(t)
	handler := mockServer.chainMiddleware(allowedOrigins, mockHandler())

	req := httptest.NewRequest("GET", "/mcp", nil)
	req.Header.Set("Origin", "http://example.com")
	// No auth credentials
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	// Should return 401 when no credentials provided
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", rec.Code)
	}
}

func TestParseAllowedOrigins_Empty(t *testing.T) {
	result := parseAllowedOrigins("")
	if len(result) != 0 {
		t.Errorf("Expected empty slice, got %v", result)
	}
}

func TestParseAllowedOrigins_Wildcard(t *testing.T) {
	result := parseAllowedOrigins("*")
	if len(result) != 1 || result[0] != "*" {
		t.Errorf("Expected [*], got %v", result)
	}
}

func TestParseAllowedOrigins_SingleOrigin(t *testing.T) {
	result := parseAllowedOrigins("http://example.com")
	if len(result) != 1 || result[0] != "http://example.com" {
		t.Errorf("Expected [http://example.com], got %v", result)
	}
}

func TestParseAllowedOrigins_MultipleOrigins(t *testing.T) {
	result := parseAllowedOrigins("http://example.com,http://localhost:3000,http://test.com")
	expected := []string{"http://example.com", "http://localhost:3000", "http://test.com"}

	if len(result) != len(expected) {
		t.Errorf("Expected %d origins, got %d", len(expected), len(result))
	}

	for i, exp := range expected {
		if result[i] != exp {
			t.Errorf("Expected origin[%d] = %q, got %q", i, exp, result[i])
		}
	}
}

func TestParseAllowedOrigins_WithSpaces(t *testing.T) {
	result := parseAllowedOrigins("http://example.com , http://localhost:3000 , http://test.com")
	expected := []string{"http://example.com", "http://localhost:3000", "http://test.com"}

	if len(result) != len(expected) {
		t.Errorf("Expected %d origins, got %d", len(expected), len(result))
	}

	for i, exp := range expected {
		if result[i] != exp {
			t.Errorf("Expected origin[%d] = %q, got %q", i, exp, result[i])
		}
	}
}

func TestPathValidationMiddleware_ValidPath(t *testing.T) {
	handler := pathValidationMiddleware()(mockHandler())

	req := httptest.NewRequest("GET", "/mcp", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200 for /mcp path, got %d", rec.Code)
	}

	if rec.Body.String() != "OK" {
		t.Errorf("Expected body 'OK', got %q", rec.Body.String())
	}
}

func TestPathValidationMiddleware_InvalidPaths(t *testing.T) {
	testCases := []struct {
		name string
		path string
	}{
		{"root path", "/"},
		{"other path", "/api"},
		{"nested path", "/mcp/test"},
		{"similar path", "/mcpserver"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			handler := pathValidationMiddleware()(mockHandler())

			req := httptest.NewRequest("GET", tc.path, nil)
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			if rec.Code != http.StatusNotFound {
				t.Errorf("Expected status 404 for path %s, got %d", tc.path, rec.Code)
			}

			expectedBody := "Not Found: This server only handles requests to /mcp\n"
			if rec.Body.String() != expectedBody {
				t.Errorf("Expected body %q, got %q", expectedBody, rec.Body.String())
			}
		})
	}
}

func TestPathValidationMiddleware_InFullChain(t *testing.T) {
	// Test that path validation happens before auth check
	// Invalid paths should return 404 without requiring auth
	allowedOrigins := []string{}
	mockServer := mockNeo4jMCPServer(t)
	handler := mockServer.chainMiddleware(allowedOrigins, mockHandler())

	req := httptest.NewRequest("GET", "/", nil)
	// No auth credentials
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	// Should return 404 for invalid path, not 401 for missing auth
	// This proves path validation happens first in the middleware chain
	if rec.Code != http.StatusNotFound {
		t.Errorf("Expected status 404 for invalid path (before auth check), got %d", rec.Code)
	}
}

func TestHTTPMetricsMiddleware_TelemetryEnabled(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockAnalyticsService := analytics_mocks.NewMockService(ctrl)
	mockDBService := db_mocks.NewMockService(ctrl)

	cfg := &config.Config{
		URI:            "bolt://databases.neo4j.io:7687", // Aura URI to test isAura detection
		Database:       "neo4j",
		TransportMode:  config.TransportModeHTTP,
		Telemetry:      true, // Enable telemetry
		HTTPTLSEnabled: true, // Enable TLS for testing
	}

	mcpServer := server.NewMCPServer("test-server", "1.0.0")
	testServer := &Neo4jMCPServer{
		MCPServer:    mcpServer,
		config:       cfg,
		dbService:    mockDBService,
		anService:    mockAnalyticsService,
		version:      "1.0.0",
		gdsInstalled: false,
	}

	// Track how many times the metrics collection is called
	dbQueryCallCount := 0
	emitEventCallCount := 0

	// Expect metrics to be collected once
	mockDBService.EXPECT().ExecuteReadQuery(gomock.Any(), "CALL dbms.components()", gomock.Any()).DoAndReturn(
		func(ctx interface{}, query string, params map[string]any) ([]*neo4j.Record, error) {
			dbQueryCallCount++
			return nil, nil
		},
	).Times(1)

	// Match on specific StartupEventInfo fields
	mockAnalyticsService.EXPECT().NewStartupEvent(
		gomock.Cond(func(x interface{}) bool {
			info, ok := x.(analytics.StartupEventInfo)
			if !ok {
				return false
			}
			// Verify expected fields
			return info.McpVersion == "1.0.0"
		}),
	).Return(analytics.TrackEvent{
		Event:      "MCP4NEO4J_MCP_STARTUP",
		Properties: map[string]interface{}{}, // Actual properties don't matter for the return
	}).Times(1)

	// Match on specific TrackEvent structure
	mockAnalyticsService.EXPECT().EmitEvent(
		gomock.Cond(func(x interface{}) bool {
			event, ok := x.(analytics.TrackEvent)
			if !ok {
				t.Errorf("EmitEvent called with non-TrackEvent: %T", x)
				return false
			}
			emitEventCallCount++

			// Verify event name matches
			if event.Event != "MCP4NEO4J_MCP_STARTUP" {
				t.Errorf("Expected event 'MCP4NEO4J_MCP_STARTUP', got '%s'", event.Event)
				return false
			}

			// The Properties will be the actual struct from analytics package
			// We can't access httpStartupProperties here since it's unexported,
			// but we can verify it's not nil and is the right type structure
			if event.Properties == nil {
				t.Errorf("Expected non-nil properties")
				return false
			}

			return true
		}),
	).Times(1)

	handler := testServer.httpMetricsMiddleware()(mockHandler())

	// Make first request
	req1 := httptest.NewRequest("GET", "/", nil)
	rec1 := httptest.NewRecorder()
	handler.ServeHTTP(rec1, req1)

	// Make second request - this should NOT trigger metrics collection again
	req2 := httptest.NewRequest("GET", "/", nil)
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req2)

	// Make third request - verify sync.Once is working
	req3 := httptest.NewRequest("GET", "/", nil)
	rec3 := httptest.NewRecorder()
	handler.ServeHTTP(rec3, req3)

	// All requests should succeed
	if rec1.Code != http.StatusOK {
		t.Errorf("Expected status 200 for first request, got %d", rec1.Code)
	}
	if rec2.Code != http.StatusOK {
		t.Errorf("Expected status 200 for second request, got %d", rec2.Code)
	}
	if rec3.Code != http.StatusOK {
		t.Errorf("Expected status 200 for third request, got %d", rec3.Code)
	}

	// Wait for the goroutine to complete
	time.Sleep(100 * time.Millisecond)

	// Verify metrics were collected exactly once (sync.Once behavior)
	if dbQueryCallCount != 1 {
		t.Errorf("Expected DB query to be called exactly once, but was called %d times", dbQueryCallCount)
	}
	if emitEventCallCount != 1 {
		t.Errorf("Expected EmitEvent to be called exactly once, but was called %d times", emitEventCallCount)
	}
}

func TestHTTPMetricsMiddleware_TelemetryDisabled(t *testing.T) {
	ctrl := gomock.NewController(t)
	mockAnalyticsService := analytics_mocks.NewMockService(ctrl)
	mockDBService := db_mocks.NewMockService(ctrl)

	cfg := &config.Config{
		URI:           "bolt://localhost:7687",
		Database:      "neo4j",
		TransportMode: config.TransportModeHTTP,
		Telemetry:     false, // Disable telemetry
	}

	mcpServer := server.NewMCPServer("test-server", "1.0.0")
	testServer := &Neo4jMCPServer{
		MCPServer:    mcpServer,
		config:       cfg,
		dbService:    mockDBService,
		anService:    mockAnalyticsService,
		version:      "1.0.0",
		gdsInstalled: false,
	}

	// No metrics should be collected
	mockDBService.EXPECT().ExecuteReadQuery(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	mockAnalyticsService.EXPECT().NewStartupEvent(gomock.Any()).Times(0)
	mockAnalyticsService.EXPECT().EmitEvent(gomock.Any()).Times(0)

	handler := testServer.httpMetricsMiddleware()(mockHandler())

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}
}
