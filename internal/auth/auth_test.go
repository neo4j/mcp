//
// Copyright (c) "Neo4j"
// Neo4j Sweden AB [http://neo4j.com]

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
// See internal/server/middleware.go, an empty bearer token can never exist in context in production.

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

func TestHasAuthCredentials(t *testing.T) {
	t.Run("with basic auth", func(t *testing.T) {
		ctx := context.Background()
		ctx = WithBasicAuth(ctx, "user", "pass")

		if !HasAuth(ctx) {
			t.Error("Expected HasAuthCredentials to return true for basic auth, got false")
		}
	})

	t.Run("with bearer token", func(t *testing.T) {
		ctx := context.Background()
		ctx = WithBearerToken(ctx, "token")

		if !HasAuth(ctx) {
			t.Error("Expected HasAuthCredentials to return true for bearer token, got false")
		}
	})

	t.Run("with no auth", func(t *testing.T) {
		ctx := context.Background()

		if HasAuth(ctx) {
			t.Error("Expected HasAuthCredentials to return false for no auth, got true")
		}
	})
}
