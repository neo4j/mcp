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
				Telemetry: "true",
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
				Telemetry: "true",
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
				Telemetry: "true",
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
				Telemetry: "true",
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
				Telemetry: "true",
				URI:       "bolt://localhost:7687",
				Username:  "neo4j",
				Password:  "password",
				Database:  "",
			},
			wantErr: false,
			errMsg:  "",
		},
		{
			name: "Invalid NEO4J_TELEMETRY type",
			cfg: &Config{
				Telemetry: "falsy",
				URI:       "bolt://localhost:7687",
				Username:  "neo4j",
				Password:  "password",
			},
			wantErr: true,
			errMsg:  "NEO4J_TELEMETRY cannot be converted to type bool",
		},
		{
			name: "Correct NEO4J_TELEMETRY type",
			cfg: &Config{
				Telemetry: "true",
				URI:       "bolt://localhost:7687",
				Username:  "neo4j",
				Password:  "password",
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

	cfg := LoadConfig()

	if cfg == nil {
		t.Error("LoadConfig() returned nil config")
		return
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

	// Verify that the returned config passes validation
	if err := cfg.Validate(); err != nil {
		t.Errorf("LoadConfig() returned config that fails validation: %v", err)
	}
}

func TestLoadConfig_MissingRequiredEnvVars(t *testing.T) {
	// Unit test: verify LoadConfig returns config with empty required fields
	// which will fail validation
	t.Setenv("NEO4J_URI", "")
	t.Setenv("NEO4J_USERNAME", "")
	t.Setenv("NEO4J_PASSWORD", "")

	cfg := LoadConfig()

	if cfg == nil {
		t.Error("LoadConfig() returned nil config")
		return
	}

	// Verify that validation fails
	err := cfg.Validate()
	if err == nil {
		t.Error("Config.Validate() expected error when required env vars are missing, got nil")
		return
	}

	// Should contain an error about required fields
	if !strings.Contains(err.Error(), "required") {
		t.Errorf("Config.Validate() error = %v, want error containing 'required'", err)
	}
}

