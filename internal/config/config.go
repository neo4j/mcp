package config

import (
	"os"
)

// Config holds the application configuration
type Config struct {
	URI      string
	Username string
	Password string
	Database string
}

// LoadConfig loads configuration from environment variables with defaults
func LoadConfig() *Config {
	return &Config{
		URI:      getEnv("NEO4J_URI", "bolt://localhost:7687"),
		Username: getEnv("NEO4J_USERNAME", "neo4j"),
		Password: getEnv("NEO4J_PASSWORD", "password"),
		Database: getEnv("NEO4J_DATABASE", "neo4j"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
