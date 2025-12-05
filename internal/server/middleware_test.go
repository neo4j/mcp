package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/neo4j/mcp/internal/auth"
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

	if rec.Header().Get("Access-Control-Max-Age") != maxAgeSeconds {
		t.Errorf("Expected Access-Control-Max-Age: %s, got %q", maxAgeSeconds, rec.Header().Get("Access-Control-Max-Age"))
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
	handler := addMiddleware(allowedOrigins, authCheckHandler(t, true, "user", "pass"))

	req := httptest.NewRequest("GET", "/", nil)
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
	handler := addMiddleware(allowedOrigins, mockHandler())

	req := httptest.NewRequest("GET", "/", nil)
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
