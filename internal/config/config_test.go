package config

import (
	"os"
	"strings"
	"testing"
)

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
				URI:      "bolt://localhost:7687",
				Username: "neo4j",
				Password: "password",
				Database: "neo4j",
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
				URI:      "",
				Username: "neo4j",
				Password: "password",
				Database: "neo4j",
			},
			wantErr: true,
			errMsg:  "Neo4j URI is required but was empty",
		},
		{
			name: "empty username",
			cfg: &Config{
				URI:      "bolt://localhost:7687",
				Username: "",
				Password: "password",
				Database: "neo4j",
			},
			wantErr: true,
			errMsg:  "Neo4j username is required but was empty",
		},
		{
			name: "empty password",
			cfg: &Config{
				URI:      "bolt://localhost:7687",
				Username: "neo4j",
				Password: "",
				Database: "neo4j",
			},
			wantErr: true,
			errMsg:  "Neo4j password is required but was empty",
		},
		{
			name: "empty database should not raise error",
			cfg: &Config{
				URI:      "bolt://localhost:7687",
				Username: "neo4j",
				Password: "password",
				Database: "",
			},
			wantErr: false,
			errMsg:  "",
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

func TestLoadConfig(t *testing.T) {
	// Test LoadConfig with current environment (whatever it is)
	// We don't modify environment variables to avoid parallel test issues
	cfg, err := LoadConfig()

	if err != nil {
		// If LoadConfig fails, it means the current environment has invalid config
		// This is fine - we just verify that validation is working
		if !strings.Contains(err.Error(), "invalid configuration") {
			t.Errorf("LoadConfig() error = %v, want error containing 'invalid configuration'", err)
		}
		if cfg != nil {
			t.Errorf("LoadConfig() expected nil config on error, got %v", cfg)
		}
		return
	}

	// If LoadConfig succeeds, verify the config is valid
	if cfg == nil {
		t.Error("LoadConfig() returned nil config without error")
		return
	}

	// Verify that the returned config passes validation
	if err := cfg.Validate(); err != nil {
		t.Errorf("LoadConfig() returned config that fails validation: %v", err)
	}

	// Verify config has reasonable default values (if env vars are not set)
	// We can't test specific values since we don't know the environment,
	// but we can verify they're not empty
	if cfg.URI == "" {
		t.Error("LoadConfig() returned empty URI")
	}
	if cfg.Username == "" {
		t.Error("LoadConfig() returned empty username")
	}
	if cfg.Password == "" {
		t.Error("LoadConfig() returned empty password")
	}
	if cfg.Database == "" {
		t.Error("LoadConfig() returned empty database")
	}
}

func TestLoadConfig_HTTPDefaults(t *testing.T) {
	// Clear HTTP-related env vars to test defaults
	originalHost := os.Getenv("MCP_HTTP_HOST")
	originalPort := os.Getenv("MCP_HTTP_PORT")
	originalPath := os.Getenv("MCP_HTTP_PATH")

	os.Unsetenv("MCP_HTTP_HOST")
	os.Unsetenv("MCP_HTTP_PORT")
	os.Unsetenv("MCP_HTTP_PATH")

	defer func() {
		if originalHost != "" {
			os.Setenv("MCP_HTTP_HOST", originalHost)
		}
		if originalPort != "" {
			os.Setenv("MCP_HTTP_PORT", originalPort)
		}
		if originalPath != "" {
			os.Setenv("MCP_HTTP_PATH", originalPath)
		}
	}()

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() failed: %v", err)
	}

	// Security: Default should be localhost-only, NOT 0.0.0.0
	if cfg.HTTPHost != "127.0.0.1" {
		t.Errorf("HTTPHost default = %v, want 127.0.0.1 (localhost-only for security)", cfg.HTTPHost)
	}

	if cfg.HTTPHost == "0.0.0.0" {
		t.Error("HTTPHost default should NOT be 0.0.0.0 (exposes to all network interfaces)")
	}

	if cfg.HTTPHost == "" {
		t.Error("HTTPHost default should NOT be empty (would bind to all interfaces)")
	}

	if cfg.HTTPPort != "8080" {
		t.Errorf("HTTPPort default = %v, want 8080", cfg.HTTPPort)
	}

	if cfg.HTTPPath != "/mcp" {
		t.Errorf("HTTPPath default = %v, want /mcp", cfg.HTTPPath)
	}
}

func TestLoadConfig_HTTPMode_SecurityValidation(t *testing.T) {
	tests := []struct {
		name               string
		httpHost           string
		auth0Domain        string
		resourceIdentifier string
		expectInsecure     bool
		description        string
	}{
		{
			name:               "localhost with no auth - less risk",
			httpHost:           "127.0.0.1",
			auth0Domain:        "",
			resourceIdentifier: "",
			expectInsecure:     true,
			description:        "Localhost without auth is insecure but lower risk",
		},
		{
			name:               "0.0.0.0 with no auth - high risk",
			httpHost:           "0.0.0.0",
			auth0Domain:        "",
			resourceIdentifier: "",
			expectInsecure:     true,
			description:        "Binding to all interfaces without auth is dangerous",
		},
		{
			name:               "0.0.0.0 with auth - acceptable",
			httpHost:           "0.0.0.0",
			auth0Domain:        "test.auth0.com",
			resourceIdentifier: "https://test-api",
			expectInsecure:     false,
			description:        "Binding to all interfaces with auth is acceptable",
		},
		{
			name:               "localhost with auth - secure",
			httpHost:           "127.0.0.1",
			auth0Domain:        "test.auth0.com",
			resourceIdentifier: "https://test-api",
			expectInsecure:     false,
			description:        "Localhost with auth is secure",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up environment
			os.Setenv("MCP_HTTP_HOST", tt.httpHost)
			os.Setenv("MCP_TRANSPORT", "http")

			if tt.auth0Domain != "" {
				os.Setenv("AUTH0_DOMAIN", tt.auth0Domain)
			} else {
				os.Unsetenv("AUTH0_DOMAIN")
			}

			if tt.resourceIdentifier != "" {
				os.Setenv("MCP_RESOURCE_IDENTIFIER", tt.resourceIdentifier)
			} else {
				os.Unsetenv("MCP_RESOURCE_IDENTIFIER")
			}

			defer func() {
				os.Unsetenv("MCP_HTTP_HOST")
				os.Unsetenv("MCP_TRANSPORT")
				os.Unsetenv("AUTH0_DOMAIN")
				os.Unsetenv("MCP_RESOURCE_IDENTIFIER")
			}()

			cfg, err := LoadConfig()
			if err != nil {
				t.Fatalf("LoadConfig() failed: %v", err)
			}

			// Verify configuration matches expectations
			if cfg.HTTPHost != tt.httpHost {
				t.Errorf("HTTPHost = %v, want %v", cfg.HTTPHost, tt.httpHost)
			}

			hasAuth := cfg.Auth0Domain != "" && cfg.ResourceIdentifier != ""
			isInsecure := !hasAuth

			if isInsecure != tt.expectInsecure {
				t.Errorf("%s: isInsecure = %v, want %v", tt.description, isInsecure, tt.expectInsecure)
			}

			// Additional security check: binding to 0.0.0.0 without auth should be flagged
			if cfg.HTTPHost == "0.0.0.0" && !hasAuth {
				t.Logf("SECURITY WARNING: Binding to 0.0.0.0 without authentication is highly insecure")
			}
		})
	}
}
