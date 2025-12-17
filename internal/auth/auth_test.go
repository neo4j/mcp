package auth

import (
	"context"
	"testing"
)

func TestWithBearerToken(t *testing.T) {
	ctx := context.Background()
	token := "test-bearer-token"

	ctx = WithBearerToken(ctx, token)

	retrieved, ok := GetBearerToken(ctx)
	if !ok {
		t.Error("Expected bearer token in context, but none found")
	}
	if retrieved != token {
		t.Errorf("Expected token %q, got %q", token, retrieved)
	}
}

func TestGetBearerToken_Missing(t *testing.T) {
	ctx := context.Background()

	token, ok := GetBearerToken(ctx)
	if ok {
		t.Error("Expected no bearer token in context, but found one")
	}

	// Verify returned token is empty when ok=false
	if token != "" {
		t.Errorf("Expected empty token when no bearer token, got %q", token)
	}
}

// Note: We don't test empty bearer tokens because the middleware (authMiddleware)
// explicitly rejects empty tokens with a 401 error before they reach the context.
// See internal/server/middleware.go lines 54-58.
// An empty bearer token can never exist in context in production.

func TestBothBearerAndBasicAuthInContext(t *testing.T) {
	// This test verifies that both bearer token and basic auth can coexist in context.
	// This test ensures they don't interfere with each other and both auth methods coexist independently in context.

	ctx := context.Background()

	// Add both auth types to context
	ctx = WithBearerToken(ctx, "test-bearer-token")
	ctx = WithBasicAuth(ctx, "testuser", "testpass")

	// Verify bearer token is present and correct
	token, hasBearerToken := GetBearerToken(ctx)
	if !hasBearerToken {
		t.Error("Expected bearer token in context")
	}
	if token != "test-bearer-token" {
		t.Errorf("Expected bearer token 'test-bearer-token', got %q", token)
	}

	// Verify basic auth is also present and correct
	user, pass, hasBasicAuth := GetBasicAuthCredentials(ctx)
	if !hasBasicAuth {
		t.Error("Expected basic auth in context")
	}
	if user != "testuser" {
		t.Errorf("Expected user 'testuser', got %q", user)
	}
	if pass != "testpass" {
		t.Errorf("Expected pass 'testpass', got %q", pass)
	}
}

func TestWithBasicAuth(t *testing.T) {
	ctx := context.Background()
	user := "testuser"
	pass := "testpass"

	ctx = WithBasicAuth(ctx, user, pass)

	retrievedUser, retrievedPass, ok := GetBasicAuthCredentials(ctx)
	if !ok {
		t.Error("Expected basic auth credentials in context, but none found")
	}
	if retrievedUser != user {
		t.Errorf("Expected user %q, got %q", user, retrievedUser)
	}
	if retrievedPass != pass {
		t.Errorf("Expected pass %q, got %q", pass, retrievedPass)
	}
}

func TestGetBasicAuthCredentials_Missing(t *testing.T) {
	ctx := context.Background()

	user, pass, ok := GetBasicAuthCredentials(ctx)
	if ok {
		t.Error("Expected no basic auth credentials in context, but found some")
	}

	// Verify returned values are empty when ok=false
	if user != "" {
		t.Errorf("Expected empty username when no credentials, got %q", user)
	}
	if pass != "" {
		t.Errorf("Expected empty password when no credentials, got %q", pass)
	}
}
