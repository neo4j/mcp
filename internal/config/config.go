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
	// TODO: support different Config types with validation https://linear.app/neo4j/issue/DEVSURF-873/support-different-types-in-the-mcp-config
	if !(c.Telemetry == "false" || c.Telemetry == "true") {
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

// LoadConfig loads configuration from environment variables with defaults
func LoadConfig() (*Config, error) {
	cfg := &Config{
		URI:       GetEnvWithDefault("NEO4J_URI", "bolt://localhost:7687"),
		Username:  GetEnvWithDefault("NEO4J_USERNAME", "neo4j"),
		Password:  GetEnvWithDefault("NEO4J_PASSWORD", "password"),
		Database:  GetEnvWithDefault("NEO4J_DATABASE", "neo4j"),
		ReadOnly:  GetEnvWithDefault("NEO4J_READ_ONLY", "false"),
		Telemetry: GetEnvWithDefault("NEO4J_TELEMETRY", "true"),
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return cfg, nil
}

func GetEnvWithDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
