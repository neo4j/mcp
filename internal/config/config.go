package config

import (
	"fmt"
	"log"
	"os"
	"strings"
)

// TransportMode defines the transport mode for the MCP server
type TransportMode string

const (
	TransportStdio TransportMode = "stdio"
	TransportHTTP  TransportMode = "http"
)

// Config holds the application configuration
type Config struct {
	URI                string
	Username           string
	Password           string
	Database           string
	TransportMode      TransportMode
	HTTPHost           string
	HTTPPort           string
	HTTPPath           string
	AllowedOrigins     []string
	Auth0Domain        string
	ResourceIdentifier string // RFC 8707: Unique identifier for this resource server
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

// LoadConfig loads configuration from environment variables with defaults
func LoadConfig() (*Config, error) {
	transportMode := TransportMode(getEnvWithDefault("MCP_TRANSPORT", string(TransportStdio)))

	// Default allowed origins for local development
	defaultOrigins := "http://localhost,http://127.0.0.1,https://localhost,https://127.0.0.1"
	allowedOriginsStr := getEnvWithDefault("MCP_ALLOWED_ORIGINS", defaultOrigins)
	allowedOrigins := parseAllowedOrigins(allowedOriginsStr)

	// Load Auth0 configuration
	auth0Domain := os.Getenv("AUTH0_DOMAIN")
	resourceIdentifier := os.Getenv("MCP_RESOURCE_IDENTIFIER")

	cfg := &Config{
		URI:                getEnvWithDefault("NEO4J_URI", "bolt://localhost:7687"),
		Username:           getEnvWithDefault("NEO4J_USERNAME", "neo4j"),
		Password:           getEnvWithDefault("NEO4J_PASSWORD", "password"),
		Database:           getEnvWithDefault("NEO4J_DATABASE", "neo4j"),
		TransportMode:      transportMode,
		HTTPHost:           getEnvWithDefault("MCP_HTTP_HOST", "127.0.0.1"),
		HTTPPort:           getEnvWithDefault("MCP_HTTP_PORT", "8080"),
		HTTPPath:           getEnvWithDefault("MCP_HTTP_PATH", "/mcp"),
		AllowedOrigins:     allowedOrigins,
		Auth0Domain:        auth0Domain,
		ResourceIdentifier: resourceIdentifier,
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	// Warn if binding to all interfaces in HTTP mode
	if cfg.TransportMode == TransportHTTP {
		if cfg.HTTPHost == "0.0.0.0" || cfg.HTTPHost == "" {
			log.Println("WARNING: HTTP server is configured to bind to all network interfaces (0.0.0.0)")
			log.Println("WARNING: For security, consider binding to localhost (127.0.0.1) instead")
			log.Println("WARNING: Set MCP_HTTP_HOST=127.0.0.1 to bind only to localhost")
		}

		// Validate Auth0 configuration for HTTP mode
		if cfg.Auth0Domain == "" || cfg.ResourceIdentifier == "" {
			log.Println("WARNING: Auth0 authentication is not configured")
			log.Println("WARNING: Set AUTH0_DOMAIN and MCP_RESOURCE_IDENTIFIER environment variables")
			log.Println("WARNING: For RFC 8707 compliance, MCP_RESOURCE_IDENTIFIER should be this server's unique URL")
			log.Println("WARNING: HTTP server will start but authentication will be disabled")
		}
	}

	return cfg, nil
}

func getEnvWithDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// parseAllowedOrigins parses a comma-separated list of allowed origins
func parseAllowedOrigins(originsStr string) []string {
	if originsStr == "" {
		return []string{}
	}

	parts := strings.Split(originsStr, ",")
	origins := make([]string, 0, len(parts))

	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			origins = append(origins, trimmed)
		}
	}

	return origins
}
