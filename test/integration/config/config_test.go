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

	// Load and validate configuration from environment variables
	cfg, err := config.LoadConfig(nil)
	if err != nil {
		t.Fatalf("Failed to load configuration: %v", err)
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
