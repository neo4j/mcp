//go:build unit

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
