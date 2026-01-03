package server

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/neo4j/mcp/internal/auth"
	"github.com/neo4j/mcp/internal/config"
)

// mockHandler is a simple handler that returns 200 OK
func mockHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})
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
	handler := basicAuthMiddleware(&config.Config{})(authCheckHandler(t, true, "testuser", "testpass"))

	req := httptest.NewRequest("GET", "/", nil)
	req.SetBasicAuth("testuser", "testpass")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}
}

func TestBasicAuthMiddleware_WithoutCredentials_ToolsCall(t *testing.T) {
	handler := basicAuthMiddleware(&config.Config{})(mockHandler())

	// tools/call requires authentication
	body := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"get-schema"}}`
	req := httptest.NewRequest("POST", "/", bytes.NewBufferString(body))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	// Should return 401 when no credentials provided for tools/call
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", rec.Code)
	}

	// Should have WWW-Authenticate header
	if rec.Header().Get("WWW-Authenticate") == "" {
		t.Error("Expected WWW-Authenticate header to be set")
	}
}

func TestBasicAuthMiddleware_WithEmptyCredentials(t *testing.T) {
	handler := basicAuthMiddleware(&config.Config{})(authCheckHandler(t, true, "", ""))

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
	cfg := &config.Config{
		TransportMode: config.TransportModeHTTP,
	}
	allowedOrigins := []string{"http://example.com"}
	handler := chainMiddleware(cfg, allowedOrigins, authCheckHandler(t, true, "user", "pass"))

	// Use tools/call to test auth is properly passed through
	body := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"get-schema"}}`
	req := httptest.NewRequest("POST", "/mcp", bytes.NewBufferString(body))
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

func TestAddMiddleware_FullChain_NoAuth_ToolsCall(t *testing.T) {
	cfg := &config.Config{
		TransportMode: config.TransportModeHTTP,
	}
	allowedOrigins := []string{"http://example.com"}
	handler := chainMiddleware(cfg, allowedOrigins, mockHandler())

	// tools/call requires authentication
	body := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"get-schema"}}`
	req := httptest.NewRequest("POST", "/mcp", bytes.NewBufferString(body))
	req.Header.Set("Origin", "http://example.com")
	// No auth credentials
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	// Should return 401 when no credentials provided for tools/call
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
	cfg := &config.Config{
		TransportMode: config.TransportModeHTTP,
	}
	allowedOrigins := []string{}
	handler := chainMiddleware(cfg, allowedOrigins, mockHandler())

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

func TestAddMiddleware_FullChain_ConfigCredentialsFallback(t *testing.T) {
	// When no request credentials, should fall back to config credentials
	cfg := &config.Config{
		TransportMode: config.TransportModeHTTP,
		Username:      "envuser",
		Password:      "envpass",
	}
	allowedOrigins := []string{"http://example.com"}
	handler := chainMiddleware(cfg, allowedOrigins, authCheckHandler(t, true, "envuser", "envpass"))

	// tools/call requires auth - should use config credentials as fallback
	body := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"get-schema"}}`
	req := httptest.NewRequest("POST", "/mcp", bytes.NewBufferString(body))
	req.Header.Set("Origin", "http://example.com")
	// No auth credentials in request - should fall back to config
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	// Should return 200 since config credentials are used as fallback
	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200 with config credentials fallback, got %d", rec.Code)
	}

	// Verify CORS headers are set
	if rec.Header().Get("Access-Control-Allow-Origin") != "http://example.com" {
		t.Errorf("Expected CORS header to be set")
	}
}

// =============================================================================
// Tests for MCP method-based authentication
// =============================================================================

func TestIsAuthRequiredForMethod(t *testing.T) {
	testCases := []struct {
		method       string
		authRequired bool
	}{
		// Methods that require auth (database access)
		{"tools/call", true},

		// Protocol methods that don't require auth
		{"initialize", false},
		{"tools/list", false},
		{"ping", false},
		{"notifications/initialized", false},
		{"notifications/cancelled", false},
		{"resources/list", false},
		{"prompts/list", false},
		{"", false}, // Empty method (malformed request)
	}

	for _, tc := range testCases {
		t.Run(tc.method, func(t *testing.T) {
			result := isAuthRequiredForMethod(tc.method)
			if result != tc.authRequired {
				t.Errorf("isAuthRequiredForMethod(%q) = %v, want %v", tc.method, result, tc.authRequired)
			}
		})
	}
}

func TestExtractMCPMethod(t *testing.T) {
	testCases := []struct {
		name           string
		body           string
		expectedMethod string
		expectError    bool
	}{
		{
			name:           "initialize method",
			body:           `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`,
			expectedMethod: "initialize",
			expectError:    false,
		},
		{
			name:           "tools/list method",
			body:           `{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}`,
			expectedMethod: "tools/list",
			expectError:    false,
		},
		{
			name:           "tools/call method",
			body:           `{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"get-schema"}}`,
			expectedMethod: "tools/call",
			expectError:    false,
		},
		{
			name:           "empty body",
			body:           "",
			expectedMethod: "",
			expectError:    false,
		},
		{
			name:           "invalid JSON",
			body:           "not json",
			expectedMethod: "",
			expectError:    false, // Returns empty method, no error
		},
		{
			name:           "JSON without method",
			body:           `{"jsonrpc":"2.0","id":1}`,
			expectedMethod: "",
			expectError:    false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/mcp", bytes.NewBufferString(tc.body))
			method, err := extractMCPMethod(req)

			if tc.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tc.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if method != tc.expectedMethod {
				t.Errorf("extractMCPMethod() = %q, want %q", method, tc.expectedMethod)
			}

			// Verify body was restored for subsequent reads
			if tc.body != "" {
				restoredBody := make([]byte, len(tc.body))
				n, _ := req.Body.Read(restoredBody)
				if string(restoredBody[:n]) != tc.body {
					t.Error("Request body was not properly restored")
				}
			}
		})
	}
}

func TestBasicAuthMiddleware_ProtocolMethodsAllowedWithoutAuth(t *testing.T) {
	testCases := []struct {
		name   string
		method string
		body   string
	}{
		{
			name:   "initialize",
			method: "initialize",
			body:   `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`,
		},
		{
			name:   "tools/list",
			method: "tools/list",
			body:   `{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}`,
		},
		{
			name:   "ping",
			method: "ping",
			body:   `{"jsonrpc":"2.0","id":3,"method":"ping"}`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			handler := basicAuthMiddleware(&config.Config{})(mockHandler())

			req := httptest.NewRequest("POST", "/mcp", bytes.NewBufferString(tc.body))
			// No auth credentials
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			// Should return 200 - protocol methods don't require auth
			if rec.Code != http.StatusOK {
				t.Errorf("Expected status 200 for %s without auth, got %d", tc.method, rec.Code)
			}
		})
	}
}

func TestBasicAuthMiddleware_ToolsCallRequiresAuth(t *testing.T) {
	handler := basicAuthMiddleware(&config.Config{})(mockHandler())

	body := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"get-schema"}}`
	req := httptest.NewRequest("POST", "/mcp", bytes.NewBufferString(body))
	// No auth credentials
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	// Should return 401 - tools/call requires auth
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401 for tools/call without auth, got %d", rec.Code)
	}

	// Should have WWW-Authenticate header
	if rec.Header().Get("WWW-Authenticate") == "" {
		t.Error("Expected WWW-Authenticate header to be set")
	}
}

func TestBasicAuthMiddleware_ToolsCallWithAuthSucceeds(t *testing.T) {
	handler := basicAuthMiddleware(&config.Config{})(authCheckHandler(t, true, "neo4j", "password"))

	body := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"get-schema"}}`
	req := httptest.NewRequest("POST", "/mcp", bytes.NewBufferString(body))
	req.SetBasicAuth("neo4j", "password")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	// Should return 200 - tools/call with valid auth
	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200 for tools/call with auth, got %d", rec.Code)
	}
}

func TestBasicAuthMiddleware_ProtocolMethodsWithAuthStoresCredentials(t *testing.T) {
	// Even for protocol methods, if credentials are provided, they should be stored
	handler := basicAuthMiddleware(&config.Config{})(authCheckHandler(t, true, "neo4j", "password"))

	body := `{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}`
	req := httptest.NewRequest("POST", "/mcp", bytes.NewBufferString(body))
	req.SetBasicAuth("neo4j", "password")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200 for tools/list with auth, got %d", rec.Code)
	}
}

func TestBasicAuthMiddleware_FullChain_ProtocolMethodWithoutAuth(t *testing.T) {
	cfg := &config.Config{
		TransportMode: config.TransportModeHTTP,
	}
	allowedOrigins := []string{}
	handler := chainMiddleware(cfg, allowedOrigins, mockHandler())

	body := `{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}`
	req := httptest.NewRequest("POST", "/mcp", bytes.NewBufferString(body))
	// No auth credentials
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	// Should return 200 - tools/list doesn't require auth
	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200 for tools/list in full chain, got %d", rec.Code)
	}
}

func TestBasicAuthMiddleware_FullChain_ToolsCallWithoutAuth(t *testing.T) {
	cfg := &config.Config{
		TransportMode: config.TransportModeHTTP,
	}
	allowedOrigins := []string{}
	handler := chainMiddleware(cfg, allowedOrigins, mockHandler())

	body := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"read-cypher"}}`
	req := httptest.NewRequest("POST", "/mcp", bytes.NewBufferString(body))
	// No auth credentials
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	// Should return 401 - tools/call requires auth
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401 for tools/call without auth in full chain, got %d", rec.Code)
	}
}
