// Copyright (c) "Neo4j"
// Neo4j Sweden AB [http://neo4j.com]

package config

import (
	"strings"
	"testing"

	"github.com/neo4j/mcp/internal/testutil"
)

// testVertexToken is a non-secret placeholder used as the embedding credential in
// EmbeddingConfig struct fixtures. It is a variable (not a string literal in each
// struct) to avoid gosec G101 false positives on hardcoded credentials in tests.
var testVertexToken = "placeholder-token"

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid config",
			cfg: &Config{
				Telemetry: true,
				ReadOnly:  false,
				URI:       "bolt://localhost:7687",
				Username:  "neo4j",
				Password:  "password",
				Database:  "neo4j",
			},
			wantErr: false,
		},
		{
			name:    "nil config",
			cfg:     nil,
			wantErr: true,
			errMsg:  "configuration is required but was nil",
		},
		{
			name: "empty URI",
			cfg: &Config{
				Telemetry: true,
				URI:       "",
				Username:  "neo4j",
				Password:  "password",
				Database:  "neo4j",
			},
			wantErr: true,
			errMsg:  "Neo4j URI is required but was empty",
		},
		{
			name: "empty username",
			cfg: &Config{
				Telemetry: true,
				URI:       "bolt://localhost:7687",
				Username:  "",
				Password:  "password",
				Database:  "neo4j",
			},
			wantErr: true,
			errMsg:  "Neo4j username is required for STDIO mode",
		},
		{
			name: "empty password",
			cfg: &Config{
				Telemetry: true,
				URI:       "bolt://localhost:7687",
				Username:  "neo4j",
				Password:  "",
				Database:  "neo4j",
			},
			wantErr: true,
			errMsg:  "Neo4j password is required for STDIO mode",
		},
		{
			name: "empty database should not raise error",
			cfg: &Config{
				Telemetry: true,
				URI:       "bolt://localhost:7687",
				Username:  "neo4j",
				Password:  "password",
				Database:  "",
			},
			wantErr: false,
			errMsg:  "",
		},
		{
			name: "credentials set for HTTP mode should raise error",
			cfg: &Config{
				Telemetry:     true,
				URI:           "bolt://localhost:7687",
				Username:      "neo4j",
				Password:      "password",
				Database:      "neo4j",
				TransportMode: TransportModeHTTP,
			},
			wantErr: true,
			errMsg:  "Neo4j username and password should not be set for HTTP transport mode; credentials are provided per-request via Basic Auth headers",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()

			if tt.wantErr {
				if err == nil {
					t.Errorf("Validate() expected error but got none")
					return
				}
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("Validate() error = %v, want error containing %v", err, tt.errMsg)
				}
				return
			}

			if err != nil {
				t.Errorf("Validate() unexpected error = %v", err)
			}
		})
	}
}

func TestLoadConfig_ValidConfig(t *testing.T) {
	// Unit test: set required env variables and verify LoadConfig works
	t.Setenv("NEO4J_TRANSPORT_MODE", "stdio")
	t.Setenv("NEO4J_URI", "bolt://localhost:7687")
	t.Setenv("NEO4J_USERNAME", "testuser")
	t.Setenv("NEO4J_PASSWORD", "testpass")
	t.Setenv("NEO4J_DATABASE", "neo4j")

	cfg, err := LoadConfig(nil)
	if err != nil {
		t.Fatalf("LoadConfig() unexpected error: %v", err)
	}

	if cfg == nil {
		t.Fatal("LoadConfig() returned nil config")
	}

	if cfg.URI != "bolt://localhost:7687" {
		t.Errorf("LoadConfig() URI = %v, want bolt://localhost:7687", cfg.URI)
	}
	if cfg.Username != "testuser" {
		t.Errorf("LoadConfig() Username = %v, want testuser", cfg.Username)
	}
	if cfg.Password != "testpass" {
		t.Errorf("LoadConfig() Password = %v, want testpass", cfg.Password)
	}
	if cfg.Database != "neo4j" {
		t.Errorf("LoadConfig() Database = %v, want neo4j", cfg.Database)
	}
}

func TestLoadConfig_DeprecatedValidConfig(t *testing.T) {
	// Unit test: set required env variables and verify LoadConfig works
	t.Setenv("NEO4J_MCP_TRANSPORT", "stdio")
	t.Setenv("NEO4J_URI", "bolt://localhost:7687")
	t.Setenv("NEO4J_USERNAME", "testuser")
	t.Setenv("NEO4J_PASSWORD", "testpass")
	t.Setenv("NEO4J_DATABASE", "neo4j")

	cfg, err := LoadConfig(nil)
	if err != nil {
		t.Fatalf("LoadConfig() unexpected error: %v", err)
	}

	if cfg == nil {
		t.Fatal("LoadConfig() returned nil config")
	}

	if cfg.URI != "bolt://localhost:7687" {
		t.Errorf("LoadConfig() URI = %v, want bolt://localhost:7687", cfg.URI)
	}
	if cfg.Username != "testuser" {
		t.Errorf("LoadConfig() Username = %v, want testuser", cfg.Username)
	}
	if cfg.Password != "testpass" {
		t.Errorf("LoadConfig() Password = %v, want testpass", cfg.Password)
	}
	if cfg.Database != "neo4j" {
		t.Errorf("LoadConfig() Database = %v, want neo4j", cfg.Database)
	}
}

func TestLoadConfig_MissingRequiredEnvVars(t *testing.T) {
	// Unit test: verify LoadConfig returns error when required env vars are missing
	t.Setenv("NEO4J_TRANSPORT_MODE", "stdio")
	t.Setenv("NEO4J_URI", "")
	t.Setenv("NEO4J_USERNAME", "")
	t.Setenv("NEO4J_PASSWORD", "")

	cfg, err := LoadConfig(nil)

	// LoadConfig should return an error because validation fails
	if err == nil {
		t.Error("LoadConfig() expected error when required env vars are missing, got nil")
		return
	}

	// Config should be nil when there's an error
	if cfg != nil {
		t.Error("LoadConfig() expected nil config when validation fails, got config")
	}

	// Should contain an error about required fields
	if !strings.Contains(err.Error(), "required") {
		t.Errorf("LoadConfig() error = %v, want error containing 'required'", err)
	}
}

func TestLoadConfig_CLIOverrides(t *testing.T) {
	// Unit test: verify CLI overrides take precedence over environment variables
	t.Setenv("NEO4J_TRANSPORT_MODE", "stdio")
	t.Setenv("NEO4J_URI", "bolt://env-host:7687")
	t.Setenv("NEO4J_USERNAME", "env-user")
	t.Setenv("NEO4J_PASSWORD", "env-pass")
	t.Setenv("NEO4J_DATABASE", "env-db")

	overrides := &CLIOverrides{
		URI:      "bolt://cli-host:7687",
		Username: "cli-user",
		Password: "cli-pass",
		Database: "cli-db",
	}

	cfg, err := LoadConfig(overrides)
	if err != nil {
		t.Fatalf("LoadConfig() unexpected error: %v", err)
	}

	// Verify CLI values override env values
	if cfg.URI != "bolt://cli-host:7687" {
		t.Errorf("LoadConfig() URI = %v, want bolt://cli-host:7687", cfg.URI)
	}
	if cfg.Username != "cli-user" {
		t.Errorf("LoadConfig() Username = %v, want cli-user", cfg.Username)
	}
	if cfg.Password != "cli-pass" {
		t.Errorf("LoadConfig() Password = %v, want cli-pass", cfg.Password)
	}
	if cfg.Database != "cli-db" {
		t.Errorf("LoadConfig() Database = %v, want cli-db", cfg.Database)
	}
}

func TestLoadConfig_PartialCLIOverrides(t *testing.T) {
	// Unit test: verify partial CLI overrides work (some from CLI, some from env)
	t.Setenv("NEO4J_TRANSPORT_MODE", "stdio")
	t.Setenv("NEO4J_URI", "bolt://env-host:7687")
	t.Setenv("NEO4J_USERNAME", "env-user")
	t.Setenv("NEO4J_PASSWORD", "env-pass")
	t.Setenv("NEO4J_DATABASE", "env-db")

	// Only override URI and Username, leave Password and Database from env
	overrides := &CLIOverrides{
		URI:      "bolt://cli-host:7687",
		Username: "cli-user",
		Password: "",
		Database: "",
	}

	cfg, err := LoadConfig(overrides)
	if err != nil {
		t.Fatalf("LoadConfig() unexpected error: %v", err)
	}

	// Verify CLI values override env values where provided
	if cfg.URI != "bolt://cli-host:7687" {
		t.Errorf("LoadConfig() URI = %v, want bolt://cli-host:7687", cfg.URI)
	}
	if cfg.Username != "cli-user" {
		t.Errorf("LoadConfig() Username = %v, want cli-user", cfg.Username)
	}
	// Verify env values are used where CLI values are empty
	if cfg.Password != "env-pass" {
		t.Errorf("LoadConfig() Password = %v, want env-pass", cfg.Password)
	}
	if cfg.Database != "env-db" {
		t.Errorf("LoadConfig() Database = %v, want env-db", cfg.Database)
	}
}

func TestLoadConfig_InvalidBooleanValues(t *testing.T) {
	// Unit test: verify invalid boolean values fall back to defaults
	t.Setenv("NEO4J_TRANSPORT_MODE", "stdio")
	t.Setenv("NEO4J_URI", "bolt://localhost:7687")
	t.Setenv("NEO4J_USERNAME", "testuser")
	t.Setenv("NEO4J_PASSWORD", "testpass")
	t.Setenv("NEO4J_TELEMETRY", "invalid-value")
	t.Setenv("NEO4J_READ_ONLY", "not-a-boolean")

	cfg, err := LoadConfig(nil)
	if err != nil {
		t.Fatalf("LoadConfig() unexpected error: %v", err)
	}

	// Invalid NEO4J_TELEMETRY should fall back to default (true)
	if cfg.Telemetry != true {
		t.Errorf("LoadConfig() Telemetry = %v, want true (default for invalid value)", cfg.Telemetry)
	}

	// Invalid NEO4J_READ_ONLY should fall back to default (false)
	if cfg.ReadOnly != false {
		t.Errorf("LoadConfig() ReadOnly = %v, want false (default for invalid value)", cfg.ReadOnly)
	}
}

func TestLoadConfig_ValidBooleanValues(t *testing.T) {
	// Unit test: verify valid boolean values are parsed correctly
	t.Setenv("NEO4J_TRANSPORT_MODE", "stdio")
	t.Setenv("NEO4J_URI", "bolt://localhost:7687")
	t.Setenv("NEO4J_USERNAME", "testuser")
	t.Setenv("NEO4J_PASSWORD", "testpass")
	t.Setenv("NEO4J_TELEMETRY", "false")
	t.Setenv("NEO4J_READ_ONLY", "true")

	cfg, err := LoadConfig(nil)
	if err != nil {
		t.Fatalf("LoadConfig() unexpected error: %v", err)
	}

	// Verify telemetry is disabled
	if cfg.Telemetry != false {
		t.Errorf("LoadConfig() Telemetry = %v, want false", cfg.Telemetry)
	}

	// Verify read-only is enabled
	if cfg.ReadOnly != true {
		t.Errorf("LoadConfig() ReadOnly = %v, want true", cfg.ReadOnly)
	}
}

func TestLoadConfig_ValidIntValue(t *testing.T) {
	// Set required env variables for basic validation to pass
	t.Setenv("NEO4J_TRANSPORT_MODE", "stdio")
	t.Setenv("NEO4J_URI", "bolt://localhost:7687")
	t.Setenv("NEO4J_USERNAME", "testuser")
	t.Setenv("NEO4J_PASSWORD", "testpass")

	t.Run("default value", func(t *testing.T) {
		// Unset the env var to test default
		t.Setenv("NEO4J_SCHEMA_SAMPLE_SIZE", "")

		cfg, err := LoadConfig(nil)
		if err != nil {
			t.Fatalf("LoadConfig() unexpected error: %v", err)
		}

		if cfg.SchemaSampleSize != 100 {
			t.Errorf("LoadConfig() SchemaSampleSize = %v, want 100", cfg.SchemaSampleSize)
		}
	})

	t.Run("value from env", func(t *testing.T) {
		t.Setenv("NEO4J_SCHEMA_SAMPLE_SIZE", "500")

		cfg, err := LoadConfig(nil)
		if err != nil {
			t.Fatalf("LoadConfig() unexpected error: %v", err)
		}

		if cfg.SchemaSampleSize != 500 {
			t.Errorf("LoadConfig() SchemaSampleSize = %v, want 500", cfg.SchemaSampleSize)
		}
	})

	t.Run("invalid value from env", func(t *testing.T) {
		t.Setenv("NEO4J_SCHEMA_SAMPLE_SIZE", "invalid")

		cfg, err := LoadConfig(nil)
		if err != nil {
			t.Fatalf("LoadConfig() unexpected error: %v", err)
		}

		// Should fall back to default
		if cfg.SchemaSampleSize != 100 {
			t.Errorf("LoadConfig() SchemaSampleSize = %v, want 100", cfg.SchemaSampleSize)
		}
	})
}

func TestConfig_Validate_TLS(t *testing.T) {
	// Generate test certificates once for all test cases
	certPath, keyPath := testutil.GenerateTestTLSCertificate(t)

	tests := []struct {
		name    string
		cfg     *Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "HTTP mode with TLS enabled and both cert files provided",
			cfg: &Config{
				URI:             "bolt://localhost:7687",
				TransportMode:   TransportModeHTTP,
				HTTPTLSEnabled:  true,
				HTTPTLSCertFile: certPath,
				HTTPTLSKeyFile:  keyPath,
			},
			wantErr: false,
		},
		{
			name: "HTTP mode with TLS enabled but missing cert file",
			cfg: &Config{
				URI:             "bolt://localhost:7687",
				TransportMode:   TransportModeHTTP,
				HTTPTLSEnabled:  true,
				HTTPTLSCertFile: "",
				HTTPTLSKeyFile:  "/path/to/key.pem",
			},
			wantErr: true,
			errMsg:  "TLS certificate file is required when TLS is enabled",
		},
		{
			name: "HTTP mode with TLS enabled but missing key file",
			cfg: &Config{
				URI:             "bolt://localhost:7687",
				TransportMode:   TransportModeHTTP,
				HTTPTLSEnabled:  true,
				HTTPTLSCertFile: "/path/to/cert.pem",
				HTTPTLSKeyFile:  "",
			},
			wantErr: true,
			errMsg:  "TLS key file is required when TLS is enabled",
		},
		{
			name: "HTTP mode with TLS disabled and no cert files",
			cfg: &Config{
				URI:             "bolt://localhost:7687",
				TransportMode:   TransportModeHTTP,
				HTTPTLSEnabled:  false,
				HTTPTLSCertFile: "",
				HTTPTLSKeyFile:  "",
			},
			wantErr: false,
		},
		{
			name: "STDIO mode with TLS enabled (should be ignored)",
			cfg: &Config{
				URI:             "bolt://localhost:7687",
				Username:        "neo4j",
				Password:        "password",
				TransportMode:   TransportModeStdio,
				HTTPTLSEnabled:  true,
				HTTPTLSCertFile: "",
				HTTPTLSKeyFile:  "",
			},
			wantErr: false, // TLS validation only applies to HTTP mode
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()

			if tt.wantErr {
				if err == nil {
					t.Errorf("Validate() expected error but got none")
					return
				}
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("Validate() error = %v, want error containing %v", err, tt.errMsg)
				}
				return
			}

			if err != nil {
				t.Errorf("Validate() unexpected error = %v", err)
			}
		})
	}
}

func TestLoadConfig_TLS(t *testing.T) {
	t.Run("TLS enabled via environment variables", func(t *testing.T) {
		// Generate test certificates dynamically
		certPath, keyPath := testutil.GenerateTestTLSCertificate(t)

		t.Setenv("NEO4J_URI", "bolt://localhost:7687")
		t.Setenv("NEO4J_TRANSPORT_MODE", "http")
		t.Setenv("NEO4J_MCP_HTTP_TLS_ENABLED", "true")
		t.Setenv("NEO4J_MCP_HTTP_TLS_CERT_FILE", certPath)
		t.Setenv("NEO4J_MCP_HTTP_TLS_KEY_FILE", keyPath)

		cfg, err := LoadConfig(nil)
		if err != nil {
			t.Fatalf("LoadConfig() unexpected error: %v", err)
		}

		if !cfg.HTTPTLSEnabled {
			t.Errorf("LoadConfig() HTTPTLSEnabled = %v, want true", cfg.HTTPTLSEnabled)
		}
		if cfg.HTTPTLSCertFile != certPath {
			t.Errorf("LoadConfig() HTTPTLSCertFile = %v, want %v", cfg.HTTPTLSCertFile, certPath)
		}
		if cfg.HTTPTLSKeyFile != keyPath {
			t.Errorf("LoadConfig() HTTPTLSKeyFile = %v, want %v", cfg.HTTPTLSKeyFile, keyPath)
		}
	})

	t.Run("TLS disabled by default", func(t *testing.T) {
		t.Setenv("NEO4J_TRANSPORT_MODE", "stdio")
		t.Setenv("NEO4J_URI", "bolt://localhost:7687")
		t.Setenv("NEO4J_USERNAME", "neo4j")
		t.Setenv("NEO4J_PASSWORD", "password")

		cfg, err := LoadConfig(nil)
		if err != nil {
			t.Fatalf("LoadConfig() unexpected error: %v", err)
		}

		if cfg.HTTPTLSEnabled {
			t.Errorf("LoadConfig() HTTPTLSEnabled = %v, want false (default)", cfg.HTTPTLSEnabled)
		}
	})

	t.Run("TLS CLI overrides environment", func(t *testing.T) {
		// Generate test certificates dynamically
		certPath, keyPath := testutil.GenerateTestTLSCertificate(t)

		t.Setenv("NEO4J_URI", "bolt://localhost:7687")
		t.Setenv("NEO4J_TRANSPORT_MODE", "http")
		t.Setenv("NEO4J_MCP_HTTP_TLS_ENABLED", "false")
		t.Setenv("NEO4J_MCP_HTTP_TLS_CERT_FILE", certPath)
		t.Setenv("NEO4J_MCP_HTTP_TLS_KEY_FILE", keyPath)

		overrides := &CLIOverrides{
			TLSEnabled: "true",
		}

		cfg, err := LoadConfig(overrides)
		if err != nil {
			t.Fatalf("LoadConfig() unexpected error: %v", err)
		}

		if !cfg.HTTPTLSEnabled {
			t.Errorf("LoadConfig() HTTPTLSEnabled = %v, want true (from CLI)", cfg.HTTPTLSEnabled)
		}
		if cfg.HTTPTLSCertFile != certPath {
			t.Errorf("LoadConfig() HTTPTLSCertFile = %v, want %v (from env)", cfg.HTTPTLSCertFile, certPath)
		}
		if cfg.HTTPTLSKeyFile != keyPath {
			t.Errorf("LoadConfig() HTTPTLSKeyFile = %v, want %v (from env)", cfg.HTTPTLSKeyFile, keyPath)
		}
	})

	t.Run("TLS validation error when missing cert file", func(t *testing.T) {
		t.Setenv("NEO4J_URI", "bolt://localhost:7687")
		t.Setenv("NEO4J_TRANSPORT_MODE", "http")
		t.Setenv("NEO4J_MCP_HTTP_TLS_ENABLED", "true")
		t.Setenv("NEO4J_MCP_HTTP_TLS_KEY_FILE", "/path/to/key.pem")

		cfg, err := LoadConfig(nil)
		if err == nil {
			t.Error("LoadConfig() expected error when TLS cert file is missing, got nil")
			return
		}
		if cfg != nil {
			t.Error("LoadConfig() expected nil config when validation fails, got config")
		}
		if !strings.Contains(err.Error(), "TLS certificate file is required") {
			t.Errorf("LoadConfig() error = %v, want error containing 'TLS certificate file is required'", err)
		}
	})

	t.Run("TLS validation error with invalid cert/key files", func(t *testing.T) {
		t.Setenv("NEO4J_URI", "bolt://localhost:7687")
		t.Setenv("NEO4J_TRANSPORT_MODE", "http")
		t.Setenv("NEO4J_MCP_HTTP_TLS_ENABLED", "true")
		t.Setenv("NEO4J_MCP_HTTP_TLS_CERT_FILE", "/nonexistent/cert.pem")
		t.Setenv("NEO4J_MCP_HTTP_TLS_KEY_FILE", "/nonexistent/key.pem")

		cfg, err := LoadConfig(nil)
		if err == nil {
			t.Error("LoadConfig() expected error with nonexistent cert/key files, got nil")
			return
		}
		if cfg != nil {
			t.Error("LoadConfig() expected nil config when validation fails, got config")
		}
		if !strings.Contains(err.Error(), "failed to load TLS certificate and key") {
			t.Errorf("LoadConfig() error = %v, want error containing 'failed to load TLS certificate and key'", err)
		}
	})
}

func TestLoadConfig_DefaultHTTPPort(t *testing.T) {
	t.Run("Default port 80 when TLS disabled", func(t *testing.T) {
		t.Setenv("NEO4J_URI", "bolt://localhost:7687")
		t.Setenv("NEO4J_TRANSPORT_MODE", "http")
		// NEO4J_MCP_HTTP_TLS_ENABLED is not set (defaults to false)

		cfg, err := LoadConfig(nil)
		if err != nil {
			t.Fatalf("LoadConfig() unexpected error: %v", err)
		}

		if cfg.HTTPPort != "80" {
			t.Errorf("LoadConfig() HTTPPort = %v, want '80' (default for HTTP)", cfg.HTTPPort)
		}
	})

	t.Run("Default port 443 when TLS enabled", func(t *testing.T) {
		certPath, keyPath := testutil.GenerateTestTLSCertificate(t)

		t.Setenv("NEO4J_URI", "bolt://localhost:7687")
		t.Setenv("NEO4J_TRANSPORT_MODE", "http")
		t.Setenv("NEO4J_MCP_HTTP_TLS_ENABLED", "true")
		t.Setenv("NEO4J_MCP_HTTP_TLS_CERT_FILE", certPath)
		t.Setenv("NEO4J_MCP_HTTP_TLS_KEY_FILE", keyPath)
		// NEO4J_MCP_HTTP_PORT is not set (should default to 443)

		cfg, err := LoadConfig(nil)
		if err != nil {
			t.Fatalf("LoadConfig() unexpected error: %v", err)
		}

		if cfg.HTTPPort != "443" {
			t.Errorf("LoadConfig() HTTPPort = %v, want '443' (default for HTTPS)", cfg.HTTPPort)
		}
	})

	t.Run("Explicit port overrides default", func(t *testing.T) {
		certPath, keyPath := testutil.GenerateTestTLSCertificate(t)

		t.Setenv("NEO4J_URI", "bolt://localhost:7687")
		t.Setenv("NEO4J_TRANSPORT_MODE", "http")
		t.Setenv("NEO4J_MCP_HTTP_TLS_ENABLED", "true")
		t.Setenv("NEO4J_MCP_HTTP_TLS_CERT_FILE", certPath)
		t.Setenv("NEO4J_MCP_HTTP_TLS_KEY_FILE", keyPath)
		t.Setenv("NEO4J_MCP_HTTP_PORT", "8443")

		cfg, err := LoadConfig(nil)
		if err != nil {
			t.Fatalf("LoadConfig() unexpected error: %v", err)
		}

		if cfg.HTTPPort != "8443" {
			t.Errorf("LoadConfig() HTTPPort = %v, want '8443' (explicitly configured)", cfg.HTTPPort)
		}
	})

	t.Run("CLI override for port takes precedence", func(t *testing.T) {
		certPath, keyPath := testutil.GenerateTestTLSCertificate(t)

		t.Setenv("NEO4J_URI", "bolt://localhost:7687")
		t.Setenv("NEO4J_TRANSPORT_MODE", "http")
		t.Setenv("NEO4J_MCP_HTTP_TLS_ENABLED", "true")
		t.Setenv("NEO4J_MCP_HTTP_TLS_CERT_FILE", certPath)
		t.Setenv("NEO4J_MCP_HTTP_TLS_KEY_FILE", keyPath)
		// Don't set NEO4J_MCP_HTTP_PORT in environment

		overrides := &CLIOverrides{
			Port: "9443",
		}

		cfg, err := LoadConfig(overrides)
		if err != nil {
			t.Fatalf("LoadConfig() unexpected error: %v", err)
		}

		if cfg.HTTPPort != "9443" {
			t.Errorf("LoadConfig() HTTPPort = %v, want '9443' (from CLI override)", cfg.HTTPPort)
		}
	})

	t.Run("CLI TLS enable changes default port", func(t *testing.T) {
		certPath, keyPath := testutil.GenerateTestTLSCertificate(t)

		t.Setenv("NEO4J_URI", "bolt://localhost:7687")
		t.Setenv("NEO4J_TRANSPORT_MODE", "http")
		t.Setenv("NEO4J_MCP_HTTP_TLS_ENABLED", "false")
		t.Setenv("NEO4J_MCP_HTTP_TLS_CERT_FILE", certPath)
		t.Setenv("NEO4J_MCP_HTTP_TLS_KEY_FILE", keyPath)
		// Don't set NEO4J_MCP_HTTP_PORT

		overrides := &CLIOverrides{
			TLSEnabled: "true",
		}

		cfg, err := LoadConfig(overrides)
		if err != nil {
			t.Fatalf("LoadConfig() unexpected error: %v", err)
		}

		if cfg.HTTPPort != "443" {
			t.Errorf("LoadConfig() HTTPPort = %v, want '443' (default for HTTPS after CLI override)", cfg.HTTPPort)
		}
	})
}

func TestLoadConfig_HTTPAllowedOrigins(t *testing.T) {
	t.Run("HTTPAllowedOrigins from environment variable", func(t *testing.T) {
		t.Setenv("NEO4J_TRANSPORT_MODE", "stdio")
		t.Setenv("NEO4J_URI", "bolt://localhost:7687")
		t.Setenv("NEO4J_USERNAME", "neo4j")
		t.Setenv("NEO4J_PASSWORD", "password")
		t.Setenv("NEO4J_MCP_HTTP_ALLOWED_ORIGINS", "https://example.com,https://example2.com")

		cfg, err := LoadConfig(nil)
		if err != nil {
			t.Fatalf("LoadConfig() unexpected error: %v", err)
		}

		if cfg.HTTPAllowedOrigins != "https://example.com,https://example2.com" {
			t.Errorf("LoadConfig() HTTPAllowedOrigins = %v, want 'https://example.com,https://example2.com'", cfg.HTTPAllowedOrigins)
		}
	})

	t.Run("HTTPAllowedOrigins with wildcard from environment variable", func(t *testing.T) {
		t.Setenv("NEO4J_TRANSPORT_MODE", "stdio")
		t.Setenv("NEO4J_URI", "bolt://localhost:7687")
		t.Setenv("NEO4J_USERNAME", "neo4j")
		t.Setenv("NEO4J_PASSWORD", "password")
		t.Setenv("NEO4J_MCP_HTTP_ALLOWED_ORIGINS", "*")

		cfg, err := LoadConfig(nil)
		if err != nil {
			t.Fatalf("LoadConfig() unexpected error: %v", err)
		}

		if cfg.HTTPAllowedOrigins != "*" {
			t.Errorf("LoadConfig() HTTPAllowedOrigins = %v, want '*'", cfg.HTTPAllowedOrigins)
		}
	})

	t.Run("HTTPAllowedOrigins empty by default", func(t *testing.T) {
		t.Setenv("NEO4J_TRANSPORT_MODE", "stdio")
		t.Setenv("NEO4J_URI", "bolt://localhost:7687")
		t.Setenv("NEO4J_USERNAME", "neo4j")
		t.Setenv("NEO4J_PASSWORD", "password")
		// Don't set NEO4J_MCP_HTTP_ALLOWED_ORIGINS

		cfg, err := LoadConfig(nil)
		if err != nil {
			t.Fatalf("LoadConfig() unexpected error: %v", err)
		}

		if cfg.HTTPAllowedOrigins != "" {
			t.Errorf("LoadConfig() HTTPAllowedOrigins = %v, want '' (empty by default)", cfg.HTTPAllowedOrigins)
		}
	})

	t.Run("HTTPAllowedOrigins CLI override takes precedence over environment", func(t *testing.T) {
		t.Setenv("NEO4J_TRANSPORT_MODE", "stdio")
		t.Setenv("NEO4J_URI", "bolt://localhost:7687")
		t.Setenv("NEO4J_USERNAME", "neo4j")
		t.Setenv("NEO4J_PASSWORD", "password")
		t.Setenv("NEO4J_MCP_HTTP_ALLOWED_ORIGINS", "https://env-example.com")

		overrides := &CLIOverrides{
			AllowedOrigins: "https://cli-example.com",
		}

		cfg, err := LoadConfig(overrides)
		if err != nil {
			t.Fatalf("LoadConfig() unexpected error: %v", err)
		}

		if cfg.HTTPAllowedOrigins != "https://cli-example.com" {
			t.Errorf("LoadConfig() HTTPAllowedOrigins = %v, want 'https://cli-example.com' (from CLI)", cfg.HTTPAllowedOrigins)
		}
	})
}

func TestLoadConfig_AuthHeaderName(t *testing.T) {
	// Default header name when not set
	t.Run("default header name", func(t *testing.T) {
		t.Setenv("NEO4J_TRANSPORT_MODE", "stdio")
		t.Setenv("NEO4J_URI", "bolt://localhost:7687")
		t.Setenv("NEO4J_USERNAME", "neo4j")
		t.Setenv("NEO4J_PASSWORD", "password")

		cfg, err := LoadConfig(nil)
		if err != nil {
			t.Fatalf("LoadConfig() unexpected error: %v", err)
		}

		if cfg.AuthHeaderName != "Authorization" {
			t.Errorf("LoadConfig() AuthHeaderName = %v, want 'Authorization' (default)", cfg.AuthHeaderName)
		}
	})

	// Custom header name from environment variable
	t.Run("custom header from env", func(t *testing.T) {
		t.Setenv("NEO4J_TRANSPORT_MODE", "stdio")
		t.Setenv("NEO4J_URI", "bolt://localhost:7687")
		t.Setenv("NEO4J_USERNAME", "neo4j")
		t.Setenv("NEO4J_PASSWORD", "password")
		t.Setenv("NEO4J_HTTP_AUTH_HEADER_NAME", "X-Test-Auth")

		cfg, err := LoadConfig(nil)
		if err != nil {
			t.Fatalf("LoadConfig() unexpected error: %v", err)
		}

		if cfg.AuthHeaderName != "X-Test-Auth" {
			t.Errorf("LoadConfig() AuthHeaderName = %v, want 'X-Test-Auth' (from env)", cfg.AuthHeaderName)
		}
	})

	// CLI override should take precedence over environment variable
	t.Run("cli override takes precedence", func(t *testing.T) {
		t.Setenv("NEO4J_TRANSPORT_MODE", "stdio")
		t.Setenv("NEO4J_URI", "bolt://localhost:7687")
		t.Setenv("NEO4J_USERNAME", "neo4j")
		t.Setenv("NEO4J_PASSWORD", "password")
		t.Setenv("NEO4J_HTTP_AUTH_HEADER_NAME", "X-Env-Auth")

		overrides := &CLIOverrides{
			AuthHeaderName: "X-CLI-Auth",
		}

		cfg, err := LoadConfig(overrides)
		if err != nil {
			t.Fatalf("LoadConfig() unexpected error: %v", err)
		}

		if cfg.AuthHeaderName != "X-CLI-Auth" {
			t.Errorf("LoadConfig() AuthHeaderName = %v, want 'X-CLI-Auth' (from CLI)", cfg.AuthHeaderName)
		}
	})

	// Whitespace-only CLI override should be rejected (validation)
	t.Run("whitespace-only cli override invalid", func(t *testing.T) {
		t.Setenv("NEO4J_TRANSPORT_MODE", "stdio")
		t.Setenv("NEO4J_URI", "bolt://localhost:7687")
		t.Setenv("NEO4J_USERNAME", "neo4j")
		t.Setenv("NEO4J_PASSWORD", "password")

		overrides := &CLIOverrides{
			AuthHeaderName: "   ", // non-empty but only whitespace -> should be trimmed to empty and cause an error
		}

		cfg, err := LoadConfig(overrides)
		if err == nil {
			t.Error("LoadConfig() expected error for whitespace-only auth header CLI override, got nil")
			_ = cfg
			return
		}

		if !strings.Contains(err.Error(), "invalid auth header name") {
			t.Errorf("LoadConfig() error = %v, want error containing 'invalid auth header name'", err)
		}
	})
}

// setBaseEnv sets the minimum env vars needed for LoadConfig to pass basic validation
// so embedding tests can focus on embedding-specific behaviour.
func setBaseEnv(t *testing.T) {
	t.Helper()
	t.Setenv("NEO4J_TRANSPORT_MODE", "stdio")
	t.Setenv("NEO4J_URI", "bolt://localhost:7687")
	t.Setenv("NEO4J_USERNAME", "testuser")
	t.Setenv("NEO4J_PASSWORD", "testpass")
}

func TestEmbeddingConfig_NoProvider(t *testing.T) {
	setBaseEnv(t)
	// Ensure embedding env vars are absent
	t.Setenv("NEO4J_EMBEDDING_PROVIDER", "")

	cfg, err := LoadConfig(nil)
	if err != nil {
		t.Fatalf("LoadConfig() unexpected error: %v", err)
	}

	emb := cfg.EmbeddingConfig()
	if emb.Provider != "" {
		t.Errorf("EmbeddingConfig().Provider = %q, want empty", emb.Provider)
	}
	if cfg.IsEmbeddingConfigured() {
		t.Error("IsEmbeddingConfigured() = true, want false when no provider set")
	}
}

func TestEmbeddingConfig_OpenAI_FullyConfigured(t *testing.T) {
	setBaseEnv(t)
	t.Setenv("NEO4J_EMBEDDING_PROVIDER", "openai")
	t.Setenv("NEO4J_EMBEDDING_MODEL", "text-embedding-3-small")
	t.Setenv("NEO4J_EMBEDDING_API_KEY", "sk-test-key")
	t.Setenv("NEO4J_EMBEDDING_DIMENSIONS", "1536")

	cfg, err := LoadConfig(nil)
	if err != nil {
		t.Fatalf("LoadConfig() unexpected error: %v", err)
	}

	emb := cfg.EmbeddingConfig()
	if emb.Provider != EmbeddingProviderOpenAI {
		t.Errorf("EmbeddingConfig().Provider = %q, want %q", emb.Provider, EmbeddingProviderOpenAI)
	}
	if emb.Model != "text-embedding-3-small" {
		t.Errorf("EmbeddingConfig().Model = %q, want 'text-embedding-3-small'", emb.Model)
	}
	if emb.APIKey != "sk-test-key" {
		t.Errorf("EmbeddingConfig().APIKey = %q, want 'sk-test-key'", emb.APIKey)
	}
	if emb.Dimensions != "1536" {
		t.Errorf("EmbeddingConfig().Dimensions = %q, want '1536'", emb.Dimensions)
	}
	if !cfg.IsEmbeddingConfigured() {
		t.Error("IsEmbeddingConfigured() = false, want true for fully configured openai")
	}
}

func TestEmbeddingConfig_AzureOpenAI_FullyConfigured(t *testing.T) {
	setBaseEnv(t)
	t.Setenv("NEO4J_EMBEDDING_PROVIDER", "azure-openai")
	t.Setenv("NEO4J_EMBEDDING_MODEL", "text-embedding-ada-002")
	t.Setenv("NEO4J_EMBEDDING_API_KEY", "azure-key")
	t.Setenv("NEO4J_EMBEDDING_AZURE_RESOURCE", "my-azure-resource")

	cfg, err := LoadConfig(nil)
	if err != nil {
		t.Fatalf("LoadConfig() unexpected error: %v", err)
	}

	emb := cfg.EmbeddingConfig()
	if emb.Provider != EmbeddingProviderAzureOpenAI {
		t.Errorf("EmbeddingConfig().Provider = %q, want %q", emb.Provider, EmbeddingProviderAzureOpenAI)
	}
	if emb.AzureResource != "my-azure-resource" {
		t.Errorf("EmbeddingConfig().AzureResource = %q, want 'my-azure-resource'", emb.AzureResource)
	}
	if !cfg.IsEmbeddingConfigured() {
		t.Error("IsEmbeddingConfigured() = false, want true for fully configured azure-openai")
	}
}

func TestEmbeddingConfig_VertexAI_FullyConfigured(t *testing.T) {
	setBaseEnv(t)
	t.Setenv("NEO4J_EMBEDDING_PROVIDER", "vertexai")
	t.Setenv("NEO4J_EMBEDDING_MODEL", "textembedding-gecko")
	t.Setenv("NEO4J_EMBEDDING_API_KEY", "vertex-token")
	t.Setenv("NEO4J_EMBEDDING_VERTEX_PROJECT", "my-gcp-project")
	t.Setenv("NEO4J_EMBEDDING_VERTEX_REGION", "us-central1")
	t.Setenv("NEO4J_EMBEDDING_VERTEX_PUBLISHER", "google")

	cfg, err := LoadConfig(nil)
	if err != nil {
		t.Fatalf("LoadConfig() unexpected error: %v", err)
	}

	emb := cfg.EmbeddingConfig()
	if emb.Provider != EmbeddingProviderVertexAI {
		t.Errorf("EmbeddingConfig().Provider = %q, want %q", emb.Provider, EmbeddingProviderVertexAI)
	}
	if emb.VertexProject != "my-gcp-project" {
		t.Errorf("EmbeddingConfig().VertexProject = %q, want 'my-gcp-project'", emb.VertexProject)
	}
	if emb.VertexRegion != "us-central1" {
		t.Errorf("EmbeddingConfig().VertexRegion = %q, want 'us-central1'", emb.VertexRegion)
	}
	if emb.VertexPublisher != "google" {
		t.Errorf("EmbeddingConfig().VertexPublisher = %q, want 'google'", emb.VertexPublisher)
	}
	if !cfg.IsEmbeddingConfigured() {
		t.Error("IsEmbeddingConfigured() = false, want true for fully configured vertexai")
	}
}

func TestEmbeddingConfig_BedrockTitan_FullyConfigured(t *testing.T) {
	setBaseEnv(t)
	t.Setenv("NEO4J_EMBEDDING_PROVIDER", "bedrock-titan")
	t.Setenv("NEO4J_EMBEDDING_MODEL", "amazon.titan-embed-text-v1")
	t.Setenv("NEO4J_EMBEDDING_AWS_ACCESS_KEY_ID", "test-access-key-id")
	t.Setenv("NEO4J_EMBEDDING_AWS_SECRET_ACCESS_KEY", "test-secret-access-key")
	t.Setenv("NEO4J_EMBEDDING_AWS_REGION", "us-east-1")

	cfg, err := LoadConfig(nil)
	if err != nil {
		t.Fatalf("LoadConfig() unexpected error: %v", err)
	}

	emb := cfg.EmbeddingConfig()
	if emb.Provider != EmbeddingProviderBedrockTitan {
		t.Errorf("EmbeddingConfig().Provider = %q, want %q", emb.Provider, EmbeddingProviderBedrockTitan)
	}
	if emb.AWSAccessKeyID != "test-access-key-id" {
		t.Errorf("EmbeddingConfig().AWSAccessKeyID = %q, want 'test-access-key-id'", emb.AWSAccessKeyID)
	}
	if emb.AWSSecretAccessKey != "test-secret-access-key" {
		t.Errorf("EmbeddingConfig().AWSSecretAccessKey = %q, want 'test-secret-access-key'", emb.AWSSecretAccessKey)
	}
	if emb.AWSRegion != "us-east-1" {
		t.Errorf("EmbeddingConfig().AWSRegion = %q, want 'us-east-1'", emb.AWSRegion)
	}
	if !cfg.IsEmbeddingConfigured() {
		t.Error("IsEmbeddingConfigured() = false, want true for fully configured bedrock-titan")
	}
}

func TestEmbeddingConfig_UnknownProvider_ValidateError(t *testing.T) {
	setBaseEnv(t)
	t.Setenv("NEO4J_EMBEDDING_PROVIDER", "unknown-provider")

	cfg, err := LoadConfig(nil)
	if err == nil {
		t.Error("LoadConfig() expected error for unknown provider, got nil")
		_ = cfg
		return
	}
	if !strings.Contains(err.Error(), "invalid NEO4J_EMBEDDING_PROVIDER") {
		t.Errorf("LoadConfig() error = %v, want error containing 'invalid NEO4J_EMBEDDING_PROVIDER'", err)
	}
}

func TestEmbeddingConfig_Validate_MissingFields(t *testing.T) {
	tests := []struct {
		name          string
		envVars       map[string]string
		wantErrSubstr string
	}{
		{
			name: "openai missing model",
			envVars: map[string]string{
				"NEO4J_EMBEDDING_PROVIDER": "openai",
				"NEO4J_EMBEDDING_API_KEY":  "sk-key",
			},
			wantErrSubstr: "NEO4J_EMBEDDING_MODEL",
		},
		{
			name: "openai missing api key",
			envVars: map[string]string{
				"NEO4J_EMBEDDING_PROVIDER": "openai",
				"NEO4J_EMBEDDING_MODEL":    "text-embedding-3-small",
			},
			wantErrSubstr: "NEO4J_EMBEDDING_API_KEY",
		},
		{
			name: "azure-openai missing model",
			envVars: map[string]string{
				"NEO4J_EMBEDDING_PROVIDER":       "azure-openai",
				"NEO4J_EMBEDDING_API_KEY":        "azure-key",
				"NEO4J_EMBEDDING_AZURE_RESOURCE": "my-resource",
			},
			wantErrSubstr: "NEO4J_EMBEDDING_MODEL",
		},
		{
			name: "azure-openai missing api key",
			envVars: map[string]string{
				"NEO4J_EMBEDDING_PROVIDER":       "azure-openai",
				"NEO4J_EMBEDDING_MODEL":          "text-embedding-ada-002",
				"NEO4J_EMBEDDING_AZURE_RESOURCE": "my-resource",
			},
			wantErrSubstr: "NEO4J_EMBEDDING_API_KEY",
		},
		{
			name: "azure-openai missing azure resource",
			envVars: map[string]string{
				"NEO4J_EMBEDDING_PROVIDER": "azure-openai",
				"NEO4J_EMBEDDING_MODEL":    "text-embedding-ada-002",
				"NEO4J_EMBEDDING_API_KEY":  "azure-key",
			},
			wantErrSubstr: "NEO4J_EMBEDDING_AZURE_RESOURCE",
		},
		{
			name: "vertexai missing model",
			envVars: map[string]string{
				"NEO4J_EMBEDDING_PROVIDER":       "vertexai",
				"NEO4J_EMBEDDING_API_KEY":        "vertex-token",
				"NEO4J_EMBEDDING_VERTEX_PROJECT": "my-project",
				"NEO4J_EMBEDDING_VERTEX_REGION":  "us-central1",
			},
			wantErrSubstr: "NEO4J_EMBEDDING_MODEL",
		},
		{
			name: "vertexai missing api key",
			envVars: map[string]string{
				"NEO4J_EMBEDDING_PROVIDER":       "vertexai",
				"NEO4J_EMBEDDING_MODEL":          "textembedding-gecko",
				"NEO4J_EMBEDDING_VERTEX_PROJECT": "my-project",
				"NEO4J_EMBEDDING_VERTEX_REGION":  "us-central1",
			},
			wantErrSubstr: "NEO4J_EMBEDDING_API_KEY",
		},
		{
			name: "vertexai missing project",
			envVars: map[string]string{
				"NEO4J_EMBEDDING_PROVIDER":      "vertexai",
				"NEO4J_EMBEDDING_MODEL":         "textembedding-gecko",
				"NEO4J_EMBEDDING_API_KEY":       "vertex-token",
				"NEO4J_EMBEDDING_VERTEX_REGION": "us-central1",
			},
			wantErrSubstr: "NEO4J_EMBEDDING_VERTEX_PROJECT",
		},
		{
			name: "vertexai missing region",
			envVars: map[string]string{
				"NEO4J_EMBEDDING_PROVIDER":       "vertexai",
				"NEO4J_EMBEDDING_MODEL":          "textembedding-gecko",
				"NEO4J_EMBEDDING_API_KEY":        "vertex-token",
				"NEO4J_EMBEDDING_VERTEX_PROJECT": "my-project",
			},
			wantErrSubstr: "NEO4J_EMBEDDING_VERTEX_REGION",
		},
		{
			name: "bedrock-titan missing model",
			envVars: map[string]string{
				"NEO4J_EMBEDDING_PROVIDER":              "bedrock-titan",
				"NEO4J_EMBEDDING_AWS_ACCESS_KEY_ID":     "AKID",
				"NEO4J_EMBEDDING_AWS_SECRET_ACCESS_KEY": "SECRET",
				"NEO4J_EMBEDDING_AWS_REGION":            "us-east-1",
			},
			wantErrSubstr: "NEO4J_EMBEDDING_MODEL",
		},
		{
			name: "bedrock-titan missing access key id",
			envVars: map[string]string{
				"NEO4J_EMBEDDING_PROVIDER":              "bedrock-titan",
				"NEO4J_EMBEDDING_MODEL":                 "amazon.titan-embed-text-v1",
				"NEO4J_EMBEDDING_AWS_SECRET_ACCESS_KEY": "SECRET",
				"NEO4J_EMBEDDING_AWS_REGION":            "us-east-1",
			},
			wantErrSubstr: "NEO4J_EMBEDDING_AWS_ACCESS_KEY_ID",
		},
		{
			name: "bedrock-titan missing secret access key",
			envVars: map[string]string{
				"NEO4J_EMBEDDING_PROVIDER":          "bedrock-titan",
				"NEO4J_EMBEDDING_MODEL":             "amazon.titan-embed-text-v1",
				"NEO4J_EMBEDDING_AWS_ACCESS_KEY_ID": "AKID",
				"NEO4J_EMBEDDING_AWS_REGION":        "us-east-1",
			},
			wantErrSubstr: "NEO4J_EMBEDDING_AWS_SECRET_ACCESS_KEY",
		},
		{
			name: "bedrock-titan missing region",
			envVars: map[string]string{
				"NEO4J_EMBEDDING_PROVIDER":              "bedrock-titan",
				"NEO4J_EMBEDDING_MODEL":                 "amazon.titan-embed-text-v1",
				"NEO4J_EMBEDDING_AWS_ACCESS_KEY_ID":     "AKID",
				"NEO4J_EMBEDDING_AWS_SECRET_ACCESS_KEY": "SECRET",
			},
			wantErrSubstr: "NEO4J_EMBEDDING_AWS_REGION",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setBaseEnv(t)
			// Clear all embedding vars first, then set test-specific ones
			for _, key := range []string{
				"NEO4J_EMBEDDING_PROVIDER",
				"NEO4J_EMBEDDING_MODEL",
				"NEO4J_EMBEDDING_API_KEY",
				"NEO4J_EMBEDDING_DIMENSIONS",
				"NEO4J_EMBEDDING_AZURE_RESOURCE",
				"NEO4J_EMBEDDING_VERTEX_PROJECT",
				"NEO4J_EMBEDDING_VERTEX_REGION",
				"NEO4J_EMBEDDING_VERTEX_PUBLISHER",
				"NEO4J_EMBEDDING_AWS_ACCESS_KEY_ID",
				"NEO4J_EMBEDDING_AWS_SECRET_ACCESS_KEY",
				"NEO4J_EMBEDDING_AWS_REGION",
			} {
				t.Setenv(key, "")
			}
			for k, v := range tt.envVars {
				t.Setenv(k, v)
			}

			cfg, err := LoadConfig(nil)
			if err == nil {
				t.Errorf("LoadConfig() expected error for %s, got nil", tt.name)
				_ = cfg
				return
			}
			if !strings.Contains(err.Error(), tt.wantErrSubstr) {
				t.Errorf("LoadConfig() error = %v, want error containing %q", err, tt.wantErrSubstr)
			}
		})
	}
}

func TestIsEmbeddingConfigured_MissingFields(t *testing.T) {
	tests := []struct {
		name string
		cfg  EmbeddingConfig
		want bool
	}{
		{
			name: "empty provider",
			cfg:  EmbeddingConfig{},
			want: false,
		},
		{
			name: "openai fully configured",
			cfg: EmbeddingConfig{
				Provider: EmbeddingProviderOpenAI,
				Model:    "text-embedding-3-small",
				APIKey:   "sk-key",
			},
			want: true,
		},
		{
			name: "openai missing model",
			cfg: EmbeddingConfig{
				Provider: EmbeddingProviderOpenAI,
				APIKey:   "sk-key",
			},
			want: false,
		},
		{
			name: "openai missing api key",
			cfg: EmbeddingConfig{
				Provider: EmbeddingProviderOpenAI,
				Model:    "text-embedding-3-small",
			},
			want: false,
		},
		{
			name: "azure-openai fully configured",
			cfg: EmbeddingConfig{
				Provider:      EmbeddingProviderAzureOpenAI,
				Model:         "text-embedding-ada-002",
				APIKey:        "azure-key",
				AzureResource: "my-resource",
			},
			want: true,
		},
		{
			name: "azure-openai missing resource",
			cfg: EmbeddingConfig{
				Provider: EmbeddingProviderAzureOpenAI,
				Model:    "text-embedding-ada-002",
				APIKey:   "azure-key",
			},
			want: false,
		},
		{
			name: "vertexai fully configured",
			cfg: EmbeddingConfig{
				Provider:      EmbeddingProviderVertexAI,
				Model:         "textembedding-gecko",
				APIKey:        testVertexToken,
				VertexProject: "my-project",
				VertexRegion:  "us-central1",
			},
			want: true,
		},
		{
			name: "vertexai missing region",
			cfg: EmbeddingConfig{
				Provider:      EmbeddingProviderVertexAI,
				Model:         "textembedding-gecko",
				APIKey:        testVertexToken,
				VertexProject: "my-project",
			},
			want: false,
		},
		{
			name: "bedrock-titan fully configured",
			cfg: EmbeddingConfig{
				Provider:           EmbeddingProviderBedrockTitan,
				Model:              "amazon.titan-embed-text-v1",
				AWSAccessKeyID:     "AKID",
				AWSSecretAccessKey: "SECRET",
				AWSRegion:          "us-east-1",
			},
			want: true,
		},
		{
			name: "bedrock-titan missing region",
			cfg: EmbeddingConfig{
				Provider:           EmbeddingProviderBedrockTitan,
				Model:              "amazon.titan-embed-text-v1",
				AWSAccessKeyID:     "AKID",
				AWSSecretAccessKey: "SECRET",
			},
			want: false,
		},
		{
			name: "unknown provider",
			cfg: EmbeddingConfig{
				Provider: "unknown",
				Model:    "some-model",
				APIKey:   "some-key",
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Config{embeddingCfg: tt.cfg}
			got := c.IsEmbeddingConfigured()
			if got != tt.want {
				t.Errorf("IsEmbeddingConfigured() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEmbeddingConfig_Validate_NoProviderNoError(t *testing.T) {
	// Validate that an existing deployment with no embedding vars passes validation
	cfg := &Config{
		URI:           "bolt://localhost:7687",
		Username:      "neo4j",
		Password:      "password",
		TransportMode: TransportModeStdio,
	}
	if err := cfg.Validate(); err != nil {
		t.Errorf("Validate() unexpected error for config with no embedding provider: %v", err)
	}
}

func TestEmbeddingConfig_Validate_UnknownProvider_DirectValidate(t *testing.T) {
	cfg := &Config{
		URI:           "bolt://localhost:7687",
		Username:      "neo4j",
		Password:      "password",
		TransportMode: TransportModeStdio,
		embeddingCfg: EmbeddingConfig{
			Provider: "not-a-real-provider",
			Model:    "some-model",
			APIKey:   "some-key",
		},
	}
	err := cfg.Validate()
	if err == nil {
		t.Fatal("Validate() expected error for unknown embedding provider, got nil")
	}
	if !strings.Contains(err.Error(), "invalid NEO4J_EMBEDDING_PROVIDER") {
		t.Errorf("Validate() error = %v, want error containing 'invalid NEO4J_EMBEDDING_PROVIDER'", err)
	}
}

func TestEmbeddingConfig_VertexPublisher_Optional(t *testing.T) {
	// VertexPublisher is optional — vertexai should be fully configured without it
	setBaseEnv(t)
	t.Setenv("NEO4J_EMBEDDING_PROVIDER", "vertexai")
	t.Setenv("NEO4J_EMBEDDING_MODEL", "textembedding-gecko")
	t.Setenv("NEO4J_EMBEDDING_API_KEY", "vertex-token")
	t.Setenv("NEO4J_EMBEDDING_VERTEX_PROJECT", "my-project")
	t.Setenv("NEO4J_EMBEDDING_VERTEX_REGION", "us-central1")
	// NEO4J_EMBEDDING_VERTEX_PUBLISHER intentionally not set

	cfg, err := LoadConfig(nil)
	if err != nil {
		t.Fatalf("LoadConfig() unexpected error: %v", err)
	}

	emb := cfg.EmbeddingConfig()
	if emb.VertexPublisher != "" {
		t.Errorf("EmbeddingConfig().VertexPublisher = %q, want '' (optional, not set)", emb.VertexPublisher)
	}
	if !cfg.IsEmbeddingConfigured() {
		t.Error("IsEmbeddingConfigured() = false, want true — VertexPublisher is optional")
	}
}

func TestEmbeddingConfig_Dimensions_Optional(t *testing.T) {
	// Dimensions is optional for all providers
	setBaseEnv(t)
	t.Setenv("NEO4J_EMBEDDING_PROVIDER", "openai")
	t.Setenv("NEO4J_EMBEDDING_MODEL", "text-embedding-3-small")
	t.Setenv("NEO4J_EMBEDDING_API_KEY", "sk-key")
	// NEO4J_EMBEDDING_DIMENSIONS intentionally not set

	cfg, err := LoadConfig(nil)
	if err != nil {
		t.Fatalf("LoadConfig() unexpected error: %v", err)
	}

	emb := cfg.EmbeddingConfig()
	if emb.Dimensions != "" {
		t.Errorf("EmbeddingConfig().Dimensions = %q, want '' (optional, not set)", emb.Dimensions)
	}
	if !cfg.IsEmbeddingConfigured() {
		t.Error("IsEmbeddingConfigured() = false, want true — Dimensions is optional")
	}
}
