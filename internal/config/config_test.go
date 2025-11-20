package config

import (
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
			errMsg:  "Neo4j username is required but was empty",
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
			errMsg:  "Neo4j password is required but was empty",
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
