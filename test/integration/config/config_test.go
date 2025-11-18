//go:build integration

package config

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/neo4j/mcp/internal/config"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

// TestLoadConfig_Neo4jConnectivity tests that the application can connect to Neo4j with the provided configuration.
// This test requires the following environment variables to be set:
//   - NEO4J_URI: Neo4j database URI
//   - NEO4J_USERNAME: Database username
//   - NEO4J_PASSWORD: Database password
//
// If any of these environment variables are missing, the test will fail with a clear error message.
// If the environment variables are set but the database is not accessible, the test will fail with a connection error.
func TestLoadConfig_Neo4jConnectivity(t *testing.T) {
	// Check for required environment variables
	requiredVars := map[string]string{
		"NEO4J_URI":      "Neo4j database URI",
		"NEO4J_USERNAME": "Database username",
		"NEO4J_PASSWORD": "Database password",
	}

	var missingVars []string
	for varName, description := range requiredVars {
		if os.Getenv(varName) == "" {
			missingVars = append(missingVars, fmt.Sprintf("%s (%s)", varName, description))
		}
	}

	if len(missingVars) > 0 {
		t.Fatalf("Integration test requires environment variables to be set:\n%v\n\nSet them with:\nexport NEO4J_URI=bolt://localhost:7687\nexport NEO4J_USERNAME=neo4j\nexport NEO4J_PASSWORD=your_password\n", missingVars)
	}

	// Load configuration from environment variables
	cfg := config.LoadConfig()

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		t.Fatalf("Configuration validation failed: %v", err)
	}

	// Attempt to connect to Neo4j
	ctx := context.Background()
	driver, err := neo4j.NewDriverWithContext(cfg.URI, neo4j.BasicAuth(cfg.Username, cfg.Password, ""))
	if err != nil {
		t.Fatalf("Failed to create Neo4j driver: %v", err)
	}
	defer driver.Close(ctx)

	// Verify connectivity to the database
	if err := driver.VerifyConnectivity(ctx); err != nil {
		t.Fatalf("Failed to verify Neo4j database connectivity at %s: %v\n\nMake sure:\n1. Neo4j is running and accessible\n2. Username and password are correct\n3. The database is available", cfg.URI, err)
	}

	// Test that we can execute a simple query
	session := driver.NewSession(ctx, neo4j.SessionConfig{AccessMode: neo4j.AccessModeRead})
	defer session.Close(ctx)

	result, err := session.Run(ctx, "RETURN 1 as num", nil)
	if err != nil {
		t.Fatalf("Failed to execute test query: %v", err)
	}

	if _, err = result.Consume(ctx); err != nil {
		t.Fatalf("Failed to consume query result: %v", err)
	}

	t.Log("Successfully connected to Neo4j")
	t.Logf("Database URI: %s", cfg.URI)
	t.Logf("Database: %s", cfg.Database)
}

// TestLoadConfig_MissingEnvVars tests that the application fails with a clear error when required env vars are missing.
func TestLoadConfig_MissingEnvVars(t *testing.T) {
	// Clear environment variables to simulate missing configuration
	originalURI := os.Getenv("NEO4J_URI")
	originalUsername := os.Getenv("NEO4J_USERNAME")
	originalPassword := os.Getenv("NEO4J_PASSWORD")

	defer func() {
		// Restore original values
		if originalURI != "" {
			os.Setenv("NEO4J_URI", originalURI)
		} else {
			os.Unsetenv("NEO4J_URI")
		}
		if originalUsername != "" {
			os.Setenv("NEO4J_USERNAME", originalUsername)
		} else {
			os.Unsetenv("NEO4J_USERNAME")
		}
		if originalPassword != "" {
			os.Setenv("NEO4J_PASSWORD", originalPassword)
		} else {
			os.Unsetenv("NEO4J_PASSWORD")
		}
	}()

	// Unset all required environment variables
	os.Unsetenv("NEO4J_URI")
	os.Unsetenv("NEO4J_USERNAME")
	os.Unsetenv("NEO4J_PASSWORD")

	// Load configuration - should succeed (returns struct with empty values)
	cfg := config.LoadConfig()

	if cfg == nil {
		t.Error("LoadConfig() returned nil, expected config struct")
		return
	}

	// Validate configuration - should fail with meaningful error
	err := cfg.Validate()
	if err == nil {
		t.Error("Config.Validate() should fail when required env vars are missing, but got no error")
		return
	}

	// Verify error message is helpful
	errMsg := err.Error()
	if errMsg == "" {
		t.Error("Error message is empty")
		return
	}

	t.Logf("Got expected error when env vars missing: %v", errMsg)
}
