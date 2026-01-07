package database

import (
	"context"
	"testing"

	"github.com/neo4j/mcp/internal/auth"
	"github.com/neo4j/mcp/internal/config"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

// applyOptions is a test helper that applies query options to a configuration
// and returns the resulting configuration for inspection.
func applyOptions(options []neo4j.ExecuteQueryConfigurationOption) *neo4j.ExecuteQueryConfiguration {
	config := &neo4j.ExecuteQueryConfiguration{}
	for _, opt := range options {
		opt(config)
	}
	return config
}

// TestBuildQueryOptions_HTTPMode_BearerToken verifies that bearer tokens
// are properly added to query options in HTTP mode.
func TestBuildQueryOptions_HTTPMode_BearerToken(t *testing.T) {
	service := &Neo4jService{
		driver:        nil, // Not needed for this test
		database:      "testdb",
		transportMode: config.TransportModeHTTP,
	}

	ctx := auth.WithBearerToken(context.Background(), "test-bearer-token")
	options := service.buildQueryOptions(ctx)

	// Apply options to configuration and inspect
	config := applyOptions(options)

	// Verify database is set
	if config.Database != "testdb" {
		t.Errorf("Expected database 'testdb', got %q", config.Database)
	}

	// Verify auth token is set
	if config.Auth == nil {
		t.Fatal("Expected auth token to be set, got nil")
	}

	// Verify it's a bearer token (BearerAuth sets a specific scheme)
	// The neo4j driver stores tokens with their scheme, we can't inspect the exact token
	// but we can verify that Auth was populated
}

// TestBuildQueryOptions_HTTPMode_BasicAuth verifies that basic auth
// is properly added to query options in HTTP mode when no bearer token is present.
func TestBuildQueryOptions_HTTPMode_BasicAuth(t *testing.T) {
	service := &Neo4jService{
		driver:        nil,
		database:      "testdb",
		transportMode: config.TransportModeHTTP,
	}

	ctx := auth.WithBasicAuth(context.Background(), "testuser", "testpass")
	options := service.buildQueryOptions(ctx)

	config := applyOptions(options)

	if config.Database != "testdb" {
		t.Errorf("Expected database 'testdb', got %q", config.Database)
	}

	if config.Auth == nil {
		t.Fatal("Expected auth token to be set, got nil")
	}
}

// TestBuildQueryOptions_HTTPMode_NoAuth verifies that when no auth is present
// in context, only the database option is added (no auth token).
func TestBuildQueryOptions_HTTPMode_NoAuth(t *testing.T) {
	service := &Neo4jService{
		driver:        nil,
		database:      "testdb",
		transportMode: config.TransportModeHTTP,
	}

	ctx := context.Background()
	options := service.buildQueryOptions(ctx)

	config := applyOptions(options)

	if config.Database != "testdb" {
		t.Errorf("Expected database 'testdb', got %q", config.Database)
	}

	// No auth in context, so Auth should be nil
	if config.Auth != nil {
		t.Errorf("Expected no auth token when no credentials in context, got %+v", config.Auth)
	}
}

// TestBuildQueryOptions_STDIOMode_NoAuthAdded verifies that in STDIO mode,
// no auth token is added to query options (driver's built-in auth is used).
func TestBuildQueryOptions_STDIOMode_NoAuthAdded(t *testing.T) {
	service := &Neo4jService{
		driver:        nil,
		database:      "testdb",
		transportMode: config.TransportModeStdio,
	}

	// Add bearer token to context (should be ignored in STDIO mode)
	ctx := auth.WithBearerToken(context.Background(), "test-token")

	options := service.buildQueryOptions(ctx)
	config := applyOptions(options)

	if config.Database != "testdb" {
		t.Errorf("Expected database 'testdb', got %q", config.Database)
	}

	// In STDIO mode, auth from context should be ignored (driver's built-in auth is used)
	if config.Auth != nil {
		t.Errorf("Expected no auth token in STDIO mode, got %+v", config.Auth)
	}
}

// TestBuildQueryOptions_STDIOMode_BasicAuthIgnored verifies that basic auth
// in context is ignored in STDIO mode.
func TestBuildQueryOptions_STDIOMode_BasicAuthIgnored(t *testing.T) {
	service := &Neo4jService{
		driver:        nil,
		database:      "testdb",
		transportMode: config.TransportModeStdio,
	}

	// Add basic auth to context (should be ignored in STDIO mode)
	ctx := auth.WithBasicAuth(context.Background(), "user", "pass")

	options := service.buildQueryOptions(ctx)
	config := applyOptions(options)

	if config.Database != "testdb" {
		t.Errorf("Expected database 'testdb', got %q", config.Database)
	}

	// In STDIO mode, auth from context should be ignored
	if config.Auth != nil {
		t.Errorf("Expected no auth token in STDIO mode, got %+v", config.Auth)
	}
}

// TestBuildQueryOptions_WithBaseOptions verifies that base options
// (like routing options) are properly included.
func TestBuildQueryOptions_WithBaseOptions(t *testing.T) {
	service := &Neo4jService{
		driver:        nil,
		database:      "testdb",
		transportMode: config.TransportModeHTTP,
	}

	ctx := auth.WithBearerToken(context.Background(), "test-token")

	// Add a base option (e.g., readers routing)
	options := service.buildQueryOptions(ctx, neo4j.ExecuteQueryWithReadersRouting())
	config := applyOptions(options)

	if config.Database != "testdb" {
		t.Errorf("Expected database 'testdb', got %q", config.Database)
	}

	if config.Auth == nil {
		t.Fatal("Expected auth token to be set, got nil")
	}

	// Verify readers routing was applied
	if config.Routing != neo4j.Read {
		t.Errorf("Expected readers routing (Read), got %v", config.Routing)
	}
}

// TestBuildQueryOptions_HTTPMode_EmptyDatabase verifies behavior with empty database name.
func TestBuildQueryOptions_HTTPMode_EmptyDatabase(t *testing.T) {
	service := &Neo4jService{
		driver:        nil,
		database:      "", // Empty database name
		transportMode: config.TransportModeHTTP,
	}

	ctx := auth.WithBearerToken(context.Background(), "test-token")
	options := service.buildQueryOptions(ctx)
	config := applyOptions(options)

	// Database should be set (even if empty)
	if config.Database != "" {
		t.Errorf("Expected empty database, got %q", config.Database)
	}

	if config.Auth == nil {
		t.Fatal("Expected auth token to be set, got nil")
	}
}
