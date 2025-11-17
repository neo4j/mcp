package logger_test

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/neo4j/mcp/internal/logger"
)

func TestDynamicLogLevelChange(t *testing.T) {
	t.Run("changing log level from info to debug shows debug logs", func(t *testing.T) {
		buf := &bytes.Buffer{}
		log := logger.New("info", "text", buf)

		// At info level, debug logs should NOT appear
		log.Debug("debug message")
		log.Info("info message")

		output := buf.String()
		if strings.Contains(output, "debug message") {
			t.Error("Expected debug message to NOT appear at info level")
		}
		if !strings.Contains(output, "info message") {
			t.Error("Expected info message to appear at info level")
		}

		// Now change to debug level
		buf.Reset()
		log.SetLevel("debug")
		log.Debug("debug message after change")
		log.Info("info message after change")

		output = buf.String()
		if !strings.Contains(output, "debug message after change") {
			t.Error("Expected debug message to appear after changing to debug level")
		}
		if !strings.Contains(output, "info message after change") {
			t.Error("Expected info message to appear after changing to debug level")
		}
	})

	t.Run("changing log level from debug to error filters info/debug logs", func(t *testing.T) {
		buf := &bytes.Buffer{}
		log := logger.New("debug", "text", buf)

		// At debug level, all logs should appear
		log.Debug("debug message")
		log.Info("info message")
		log.Error("error message")

		output := buf.String()
		if !strings.Contains(output, "debug message") {
			t.Error("Expected debug message to appear at debug level")
		}
		if !strings.Contains(output, "info message") {
			t.Error("Expected info message to appear at debug level")
		}
		if !strings.Contains(output, "error message") {
			t.Error("Expected error message to appear at debug level")
		}

		// Now change to error level
		buf.Reset()
		log.SetLevel("error")
		log.Debug("debug after error level")
		log.Info("info after error level")
		log.Error("error after error level")

		output = buf.String()
		if strings.Contains(output, "debug after error level") {
			t.Error("Expected debug message to NOT appear at error level")
		}
		if strings.Contains(output, "info after error level") {
			t.Error("Expected info message to NOT appear at error level")
		}
		if !strings.Contains(output, "error after error level") {
			t.Error("Expected error message to appear at error level")
		}
	})

	t.Run("log level strings are case insensitive", func(t *testing.T) {
		buf := &bytes.Buffer{}
		log := logger.New("info", "text", buf)

		buf.Reset()
		log.SetLevel("DEBUG")
		log.Debug("debug message uppercase")
		output := buf.String()
		if !strings.Contains(output, "debug message uppercase") {
			t.Error("Expected DEBUG (uppercase) to change log level to debug")
		}

		buf.Reset()
		log.SetLevel("Error")
		log.Error("error message mixed case")
		log.Info("info should not appear")
		output = buf.String()
		if !strings.Contains(output, "error message mixed case") {
			t.Error("Expected Error (mixed case) to change log level to error")
		}
		if strings.Contains(output, "info should not appear") {
			t.Error("Expected info to NOT appear at error level")
		}
	})

	t.Run("all log levels can be set dynamically", func(t *testing.T) {
		levels := []string{"debug", "info", "notice", "warn", "warning", "error", "critical", "alert", "emergency"}

		for _, lvl := range levels {
			buf := &bytes.Buffer{}
			log := logger.New("debug", "text", buf)

			log.SetLevel(lvl)
			log.Debug("test debug")
			log.Info("test info")
			log.Error("test error")

			// Just verify SetLevel doesn't panic
			if t.Failed() {
				t.Errorf("SetLevel(%q) caused test to fail", lvl)
			}
		}
	})

	t.Run("json format with dynamic log level changes", func(t *testing.T) {
		buf := &bytes.Buffer{}
		log := logger.New("info", "json", buf)

		log.Info("info message")
		output := buf.String()

		// Validate the output is valid JSON with expected fields
		var logEntry map[string]any
		if err := json.Unmarshal([]byte(output), &logEntry); err != nil {
			t.Errorf("Expected valid JSON output, got: %v (output: %s)", err, output)
		}
		if _, hasLevel := logEntry["level"]; !hasLevel {
			t.Error("Expected JSON output to contain 'level' field")
		}
		if msg, hasMsg := logEntry["msg"]; !hasMsg || msg != "info message" {
			t.Error("Expected JSON output to contain 'msg' field with 'info message'")
		}

		// Change to debug
		buf.Reset()
		log.SetLevel("debug")
		log.Debug("debug message")

		output = buf.String()
		logEntry = make(map[string]any)
		if err := json.Unmarshal([]byte(output), &logEntry); err != nil {
			t.Errorf("Expected valid JSON output after level change, got: %v (output: %s)", err, output)
		}
		if msg, hasMsg := logEntry["msg"]; !hasMsg || msg != "debug message" {
			t.Error("Expected JSON output to contain 'msg' field with 'debug message'")
		}
		if level, hasLevel := logEntry["level"]; !hasLevel || level != "DEBUG" {
			t.Error("Expected JSON output to contain 'level' field with 'DEBUG'")
		}
	})
}

func TestRedactionLogic(t *testing.T) {
	t.Run("sensitive keys are redacted", func(t *testing.T) {
		sensitiveFields := map[string]string{
			"password":       "my-secret-password",
			"token":          "bearer-token-123",
			"api_key":        "sk-1234567890",
			"secret":         "super-secret-value",
			"auth_token":     "auth-token-xyz",
			"encryption_key": "key-encryption-123",
			"tls_key":        "tls-key-data",
			"certificate":    "cert-data",
			"ca_cert":        "ca-cert-data",
			"ssl_cert":       "ssl-cert-data",
			"uri":            "bolt://user:pass@localhost:7687",
			"address":        "192.168.1.1",
			"server_address": "server.example.com",
			"host":           "localhost",
			"bolt_uri":       "bolt://localhost:7687",
			"path":           "/sensitive/path",
			"directory":      "/var/lib/sensitive",
			"backup_location": "/backup/location",
		}

		for key, sensitiveValue := range sensitiveFields {
			buf := &bytes.Buffer{}
			log := logger.New("info", "text", buf)

			log.Info("test message", key, sensitiveValue)
			output := buf.String()

			if strings.Contains(output, sensitiveValue) {
				t.Errorf("Expected %q to be redacted, but found value in output: %s", key, output)
			}
			if !strings.Contains(output, "[REDACTED]") {
				t.Errorf("Expected [REDACTED] marker for %q in output: %s", key, output)
			}
		}
	})

	t.Run("sensitive keys are redacted in JSON format", func(t *testing.T) {
		buf := &bytes.Buffer{}
		log := logger.New("info", "json", buf)

		log.Info("connection attempt",
			"password", "secret123",
			"token", "abc-def-ghi",
			"host", "db.example.com",
			"api_key", "key-xyz")

		output := buf.String()
		var logEntry map[string]any
		if err := json.Unmarshal([]byte(output), &logEntry); err != nil {
			t.Fatalf("Expected valid JSON output, got error: %v", err)
		}

		// Check that all sensitive fields are redacted
		if password, exists := logEntry["password"]; exists && password != "[REDACTED]" {
			t.Errorf("Expected password to be [REDACTED], got: %v", password)
		}
		if token, exists := logEntry["token"]; exists && token != "[REDACTED]" {
			t.Errorf("Expected token to be [REDACTED], got: %v", token)
		}
		if host, exists := logEntry["host"]; exists && host != "[REDACTED]" {
			t.Errorf("Expected host to be [REDACTED], got: %v", host)
		}
		if apiKey, exists := logEntry["api_key"]; exists && apiKey != "[REDACTED]" {
			t.Errorf("Expected api_key to be [REDACTED], got: %v", apiKey)
		}
	})

	t.Run("non-sensitive keys are not redacted", func(t *testing.T) {
		buf := &bytes.Buffer{}
		log := logger.New("info", "text", buf)

		log.Info("user action",
			"user_id", "12345",
			"action", "login",
			"timestamp", "2024-01-01T00:00:00Z",
			"region", "us-east-1")

		output := buf.String()

		// Non-sensitive values should appear in output
		if !strings.Contains(output, "12345") {
			t.Error("Expected non-sensitive value user_id to appear in output")
		}
		if !strings.Contains(output, "login") {
			t.Error("Expected non-sensitive value action to appear in output")
		}
		if !strings.Contains(output, "us-east-1") {
			t.Error("Expected non-sensitive value region to appear in output")
		}
	})

	t.Run("case-insensitive redaction for sensitive keys", func(t *testing.T) {
		caseVariations := []string{
			"PASSWORD",
			"Password",
			"PaSsWoRd",
			"TOKEN",
			"Token",
			"API_KEY",
			"Api_Key",
		}

		for _, keyVariation := range caseVariations {
			buf := &bytes.Buffer{}
			log := logger.New("info", "text", buf)

			log.Info("test", keyVariation, "sensitive-value")
			output := buf.String()

			if strings.Contains(output, "sensitive-value") {
				t.Errorf("Expected %q (case variation) to be redacted, but found value in output: %s", keyVariation, output)
			}
			if !strings.Contains(output, "[REDACTED]") {
				t.Errorf("Expected [REDACTED] marker for %q in output: %s", keyVariation, output)
			}
		}
	})

	t.Run("mixed sensitive and non-sensitive fields", func(t *testing.T) {
		buf := &bytes.Buffer{}
		log := logger.New("info", "json", buf)

		log.Info("database connection",
			"host", "localhost",
			"port", "7687",
			"database", "neo4j",
			"username", "neo4j",
			"password", "secret123")

		output := buf.String()
		var logEntry map[string]any
		if err := json.Unmarshal([]byte(output), &logEntry); err != nil {
			t.Fatalf("Expected valid JSON output, got error: %v", err)
		}

		// Sensitive fields should be redacted
		if password, exists := logEntry["password"]; !exists || password != "[REDACTED]" {
			t.Errorf("Expected password to be [REDACTED], got: %v", password)
		}

		// Non-sensitive fields should not be redacted
		if database, exists := logEntry["database"]; !exists || database != "neo4j" {
			t.Errorf("Expected database to be 'neo4j', got: %v", database)
		}
		if portVal, exists := logEntry["port"]; !exists || portVal != "[REDACTED]" {
			t.Errorf("Expected port to be [REDACTED] (sensitive field), got: %v", portVal)
		}
	})

	t.Run("isSensitiveKey function works correctly", func(t *testing.T) {
		testCases := []struct {
			key        string
			shouldMask bool
		}{
			// Sensitive keys
			{"password", true},
			{"Password", true},
			{"PASSWORD", true},
			{"token", true},
			{"api_key", true},
			{"secret", true},
			{"auth_token", true},
			{"encryption_key", true},
			{"tls_key", true},
			{"certificate", true},
			{"ca_cert", true},
			{"ssl_cert", true},
			{"uri", true},
			{"address", true},
			{"server_address", true},
			{"host", true},
			{"bolt_uri", true},
			{"path", true},
			{"directory", true},
			{"backup_location", true},

			// Non-sensitive keys
			{"user_id", false},
			{"action", false},
			{"timestamp", false},
			{"region", false},
			{"database", false},
			{"username", false},
			{"msg", false},
			{"level", false},
		}

		for _, tc := range testCases {
			result := logger.IsSensitiveKey(tc.key)
			if result != tc.shouldMask {
				t.Errorf("isSensitiveKey(%q) = %v, expected %v", tc.key, result, tc.shouldMask)
			}
		}
	})
}
