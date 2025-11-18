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

// LoadConfig loads configuration from environment variables without validation
func LoadConfig() *Config {
	return &Config{
		URI:       GetEnv("NEO4J_URI"),
		Username:  GetEnv("NEO4J_USERNAME"),
		Password:  GetEnv("NEO4J_PASSWORD"),
		Database:  GetEnvWithDefault("NEO4J_DATABASE", "neo4j"),
		ReadOnly:  GetEnvWithDefault("NEO4J_READ_ONLY", "false"),
		Telemetry: GetEnvWithDefault("NEO4J_TELEMETRY", "true"),
	}
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
