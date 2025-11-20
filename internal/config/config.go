package config

import (
	"fmt"
	"os"
)

// Config holds the application configuration
type Config struct {
	URI       string
	Username  string
	Password  string
	Database  string
	ReadOnly  string // If true, disables write tools
	Telemetry string // if false, disables telemetry
}

// Validate validates the configuration and returns an error if invalid
func (c *Config) Validate() error {
	if c == nil {
		return fmt.Errorf("configuration is required but was nil")
	}

	if c.Telemetry != "false" && c.Telemetry != "true" {
		return fmt.Errorf("%s cannot be converted to type %s", "NEO4J_TELEMETRY", "bool")
	}

	validations := []struct {
		value string
		name  string
	}{
		{c.URI, "Neo4j URI"},
		{c.Username, "Neo4j username"},
		{c.Password, "Neo4j password"},
	}

	for _, v := range validations {
		if v.value == "" {
			return fmt.Errorf("%s is required but was empty", v.name)
		}
	}

	return nil
}

// CLIOverrides holds optional configuration values from CLI flags
type CLIOverrides struct {
	URI       string
	Username  string
	Password  string
	Database  string
	ReadOnly  string
	Telemetry string
}

// LoadConfig loads configuration from environment variables, applies CLI overrides, and validates.
// CLI flag values take precedence over environment variables.
// Returns an error if required configuration is missing or invalid.
func LoadConfig(cliOverrides *CLIOverrides) (*Config, error) {
	cfg := &Config{
		URI:       GetEnv("NEO4J_URI"),
		Username:  GetEnv("NEO4J_USERNAME"),
		Password:  GetEnv("NEO4J_PASSWORD"),
		Database:  GetEnvWithDefault("NEO4J_DATABASE", "neo4j"),
		ReadOnly:  GetEnvWithDefault("NEO4J_READ_ONLY", "false"),
		Telemetry: GetEnvWithDefault("NEO4J_TELEMETRY", "true"),
	}

	// Apply CLI overrides if provided
	if cliOverrides != nil {
		if cliOverrides.URI != "" {
			cfg.URI = cliOverrides.URI
		}
		if cliOverrides.Username != "" {
			cfg.Username = cliOverrides.Username
		}
		if cliOverrides.Password != "" {
			cfg.Password = cliOverrides.Password
		}
		if cliOverrides.Database != "" {
			cfg.Database = cliOverrides.Database
		}
		if cliOverrides.ReadOnly != "" {
			cfg.ReadOnly = cliOverrides.ReadOnly
		}
		if cliOverrides.Telemetry != "" {
			cfg.Telemetry = cliOverrides.Telemetry
		}
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// GetEnv returns the value of an environment variable or empty string if not set
func GetEnv(key string) string {
	return os.Getenv(key)
}

// GetEnvWithDefault returns the value of an environment variable or a default value
func GetEnvWithDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
