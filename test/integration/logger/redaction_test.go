//go:build integration

package logger_test

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/neo4j/mcp/internal/logger"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

// TestRedactionWithNeo4jConnection tests redaction with real Neo4j connection scenarios
func TestRedactionWithNeo4jConnection(t *testing.T) {
	// Get Neo4j connection details from environment
	uri := os.Getenv("NEO4J_URI")
	username := os.Getenv("NEO4J_USERNAME")
	password := os.Getenv("NEO4J_PASSWORD")
	database := os.Getenv("NEO4J_DATABASE")

	if uri == "" || username == "" || password == "" {
		t.Skip("Neo4j environment variables not set. Set NEO4J_URI, NEO4J_USERNAME, NEO4J_PASSWORD to run integration tests")
	}

	t.Run("logs with real Neo4j credentials are redacted", func(t *testing.T) {
		buf := &bytes.Buffer{}
		log := logger.New("debug", "json", buf)

		// Log actual connection parameters
		log.Info("connecting to Neo4j",
			"uri", uri,
			"username", username,
			"password", password,
			"database", database,
			"timeout", "30s")

		output := buf.String()
		var logEntry map[string]any
		if err := json.Unmarshal([]byte(output), &logEntry); err != nil {
			t.Fatalf("Expected valid JSON output, got error: %v", err)
		}

		// Sensitive fields should be redacted
		if uriVal, exists := logEntry["uri"]; exists && uriVal != "[REDACTED]" {
			t.Errorf("Expected uri to be [REDACTED], but found actual value: %v", uriVal)
		}
		if pwdVal, exists := logEntry["password"]; exists && pwdVal != "[REDACTED]" {
			t.Errorf("Expected password to be [REDACTED], but found actual value: %v", pwdVal)
		}

		// Non-sensitive fields should not be redacted
		if dbVal, exists := logEntry["database"]; !exists || dbVal != database {
			t.Errorf("Expected database to be %q, got: %v", database, dbVal)
		}
		if timeoutVal, exists := logEntry["timeout"]; !exists || timeoutVal != "30s" {
			t.Errorf("Expected timeout to be '30s', got: %v", timeoutVal)
		}
	})

	t.Run("Neo4j connection errors don't leak credentials", func(t *testing.T) {
		buf := &bytes.Buffer{}
		log := logger.New("debug", "text", buf)

		// Test 1: Log real credentials with error
		log.Error("neo4j connection failed",
			"uri", uri,
			"username", username,
			"password", password,
			"error", "authentication failed")

		output := buf.String()

		// Verify sensitive values don't appear as unredacted values in logs
		// Check that password value is followed by [REDACTED] not the actual password
		if !strings.Contains(output, "password=[REDACTED]") {
			t.Error("Expected password to be [REDACTED] in error log")
		}

		// For URI, check it's redacted (URI might be complex, so check for pattern)
		if !strings.Contains(output, "uri=[REDACTED]") {
			t.Error("Expected uri to be [REDACTED] in error log")
		}

		// But the error message should still be there
		if !strings.Contains(output, "authentication failed") {
			t.Error("Expected error message to be in logs")
		}

		// Test 2: Log different password attempt
		buf.Reset()
		wrongPassword := "wrong-password-12345"
		log.Error("authentication retry",
			"password", wrongPassword,
			"attempt", "2")

		output = buf.String()
		// Check that the specific password value is not in the output
		// It should be [REDACTED] instead
		if strings.Contains(output, wrongPassword) {
			t.Error("Wrong password attempt was leaked in error log")
		}
		if !strings.Contains(output, "password=[REDACTED]") {
			t.Error("Expected password to be [REDACTED] for wrong password attempt")
		}
	})

	t.Run("actual Neo4j driver connection logs redaction", func(t *testing.T) {
		buf := &bytes.Buffer{}
		log := logger.New("debug", "json", buf)

		// Create actual Neo4j driver and log its initialization
		driver, err := neo4j.NewDriverWithContext(uri, neo4j.BasicAuth(username, password, ""))
		if err != nil {
			// Even if connection fails, test that we log it safely
			log.Error("failed to create neo4j driver",
				"uri", uri,
				"username", username,
				"error", err.Error())

			output := buf.String()
			var logEntry map[string]any
			if err := json.Unmarshal([]byte(output), &logEntry); err != nil {
				t.Fatalf("Expected valid JSON output, got error: %v", err)
			}

			// Verify credentials are redacted
			if uriVal, exists := logEntry["uri"]; exists && uriVal != "[REDACTED]" {
				t.Errorf("Expected uri to be [REDACTED] in error log, got: %v", uriVal)
			}
			return
		}
		defer driver.Close(context.Background())

		// Connection succeeded, log it
		log.Info("neo4j driver created successfully",
			"uri", uri,
			"username", username,
			"database", database)

		output := buf.String()
		var logEntry map[string]any
		if err := json.Unmarshal([]byte(output), &logEntry); err != nil {
			t.Fatalf("Expected valid JSON output, got error: %v", err)
		}

		// Verify uri is redacted
		if uriVal, exists := logEntry["uri"]; exists && uriVal != "[REDACTED]" {
			t.Errorf("Expected uri to be [REDACTED] in success log, got: %v", uriVal)
		}
	})

	t.Run("multiple connection attempts with various sensitive fields", func(t *testing.T) {
		buf := &bytes.Buffer{}
		log := logger.New("info", "json", buf)

		// Simulate multiple connection attempts with different sensitive fields
		attempts := []map[string]string{
			{
				"attempt": "1",
				"uri":     "bolt://localhost:7687",
				"host":    "localhost",
				"port":    "7687",
				"token":   "bearer-token-xyz",
			},
			{
				"attempt":    "2",
				"address":    "192.168.1.100:7687",
				"api_key":    "sk-secret-key-123",
				"secret":     "some-secret-value",
				"auth_token": "auth-xyz-789",
			},
			{
				"attempt":  "3",
				"bolt_uri": uri,
				"password": "pwd-attempt-3",
			},
		}

		for _, attempt := range attempts {
			buf.Reset()
			log.Info("connection attempt", "attempt", attempt["attempt"])
			for key, value := range attempt {
				if key != "attempt" {
					log.Info("connection parameter", key, value)
				}
			}

			output := buf.String()
			// Verify no actual sensitive values appear in any log line
			for key, value := range attempt {
				if strings.Contains(output, value) && key != "attempt" {
					t.Errorf("Sensitive value for key %q was not redacted: %s", key, value)
				}
			}
		}
	})
}
