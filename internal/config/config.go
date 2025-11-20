package config

import (
	"fmt"
	"log"
	"os"
	"strconv"
)

// Config holds the application configuration
type Config struct {
	URI       string
	Username  string
	Password  string
	Database  string
	ReadOnly  bool // If true, disables write tools
	Telemetry bool // If false, disables telemetry
}

// Validate validates the configuration and returns an error if invalid
func (c *Config) Validate() error {
	if c == nil {
		return fmt.Errorf("configuration is required but was nil")
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
		ReadOnly:  ParseBool(GetEnv("NEO4J_READ_ONLY"), false),
		Telemetry: ParseBool(GetEnv("NEO4J_TELEMETRY"), true),
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
			cfg.ReadOnly = ParseBool(cliOverrides.ReadOnly, false)
		}
		if cliOverrides.Telemetry != "" {
			cfg.Telemetry = ParseBool(cliOverrides.Telemetry, true)
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

// ParseBool parses a string to bool using strconv.ParseBool.
// Returns the default value if the string is empty or invalid.
// Logs a warning if the value is non-empty but invalid.
// Accepts: "1", "t", "T", "true", "True", "TRUE" for true
//          "0", "f", "F", "false", "False", "FALSE" for false
func ParseBool(value string, defaultValue bool) bool {
	if value == "" {
		return defaultValue
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		log.Printf("Warning: Invalid boolean value %q, using default: %v", value, defaultValue)
		return defaultValue
	}
	return parsed
}
