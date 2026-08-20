// Copyright (c) "Neo4j"
// Neo4j Sweden AB [http://neo4j.com]

package config

import (
	"crypto/tls"
	"fmt"
	"log"
	"os"
	"slices"
	"strconv"
	"strings"
)

type TransportMode string

const (
	// DefaultSchemaSampleSize is the default number of nodes to sample per label when inferring schema
	DefaultSchemaSampleSize   int32         = 100
	TransportModeStdio        TransportMode = "stdio"
	TransportModeHTTP         TransportMode = "http"
	DeprecatedVariableMessage string        = "Warning: deprecated %s %q; use %q instead. Support will be removed in v2.\n"
)

// ValidTransportModes defines the allowed transport mode values
var ValidTransportModes = []TransportMode{TransportModeStdio, TransportModeHTTP}

// Config holds the application configuration
type Config struct {
	URI                           string
	Username                      string
	Password                      string // #nosec G117
	Database                      string
	ReadOnly                      bool // If true, disables write tools
	Telemetry                     bool // If false, disables telemetry
	LogLevel                      string
	LogFormat                     string
	SchemaSampleSize              int32
	TransportMode                 TransportMode // MCP Transport mode (e.g., "stdio", "http")
	HTTPPort                      string        // HTTP server port (default: "443" with TLS, "80" without TLS)
	HTTPHost                      string        // HTTP server host (default: "127.0.0.1")
	HTTPAllowedOrigins            string        // Comma-separated list of allowed CORS origins (optional, "*" for all)
	HTTPTLSEnabled                bool          // If true, enables TLS/HTTPS for HTTP server (default: false)
	HTTPTLSCertFile               string        // Path to TLS certificate file (required if HTTPTLSEnabled is true)
	HTTPTLSKeyFile                string        // Path to TLS private key file (required if HTTPTLSEnabled is true)
	AuthHeaderName                string        // HTTP header name to read auth credentials from (default: "Authorization")
	AllowUnauthenticatedPing      bool          // If true, allows unauthenticated ping health checks in HTTP mode
	AllowUnauthenticatedToolsList bool          // If true, allows unauthenticated tools list in HTTP mode
}

// Validate validates the configuration and returns an error if invalid
func (c *Config) Validate() error {
	if c == nil {
		return fmt.Errorf("configuration is required but was nil")
	}

	// URI is always required
	if c.URI == "" {
		return fmt.Errorf("Neo4j URI is required but was empty")
	}

	// Default to stdio if not provided (maintains backward compatibility with tests constructing Config directly)
	if c.TransportMode == "" {
		c.TransportMode = TransportModeStdio
	}

	// Validate transport mode
	if !slices.Contains(ValidTransportModes, c.TransportMode) {
		return fmt.Errorf("invalid transport mode '%s', must be one of %v", c.TransportMode, ValidTransportModes)
	}

	// For STDIO mode, require username and password from environment
	// For HTTP mode, credentials come from per-request Basic Auth headers
	if c.TransportMode == TransportModeStdio {
		if c.Username == "" {
			return fmt.Errorf("Neo4j username is required for STDIO mode")
		}
		if c.Password == "" {
			return fmt.Errorf("Neo4j password is required for STDIO mode")
		}
	} else if c.Username != "" || c.Password != "" {
		return fmt.Errorf("Neo4j username and password should not be set for HTTP transport mode; credentials are provided per-request via Basic Auth headers")
	}

	// For HTTP mode with TLS enabled, require certificate and key files
	if c.TransportMode == TransportModeHTTP && c.HTTPTLSEnabled {
		if c.HTTPTLSCertFile == "" {
			return fmt.Errorf("TLS certificate file is required when TLS is enabled (set NEO4J_MCP_HTTP_TLS_CERT_FILE)")
		}
		if c.HTTPTLSKeyFile == "" {
			return fmt.Errorf("TLS key file is required when TLS is enabled (set NEO4J_MCP_HTTP_TLS_KEY_FILE)")
		}

		// Validate that certificate and key files exist and are valid
		// This provides early, clear error messages before attempting to start the server
		if _, err := tls.LoadX509KeyPair(c.HTTPTLSCertFile, c.HTTPTLSKeyFile); err != nil {
			return fmt.Errorf("failed to load TLS certificate and key: %w", err)
		}
	}

	return nil
}

// CLIOverrides holds optional configuration values from CLI flags
type CLIOverrides struct {
	URI                           string
	Username                      string
	Password                      string // #nosec G117
	Database                      string
	ReadOnly                      string
	Telemetry                     string
	TransportMode                 string
	Port                          string
	Host                          string
	AllowedOrigins                string
	TLSEnabled                    string
	TLSCertFile                   string
	TLSKeyFile                    string
	AuthHeaderName                string
	AllowUnauthenticatedPing      string
	AllowUnauthenticatedToolsList string
}

// LoadConfig loads configuration from environment variables, applies CLI overrides, and validates.
// CLI flag values take precedence over environment variables.
// Returns an error if required configuration is missing or invalid.
func LoadConfig(cliOverrides *CLIOverrides) (*Config, error) {
	warnOnDeprecatedUsage()

	cfg := &Config{
		URI:                           GetEnvWithAliases("NEO4J_MCP_URI", "NEO4J_URI"),
		Username:                      GetEnvWithAliases("NEO4J_MCP_USERNAME", "NEO4J_USERNAME"),
		Password:                      GetEnvWithAliases("NEO4J_MCP_PASSWORD", "NEO4J_PASSWORD"),
		Database:                      GetEnvWithAliasesDefault("NEO4J_MCP_DATABASE", "neo4j", "NEO4J_DATABASE"),
		ReadOnly:                      ParseBool(GetEnvWithAliases("NEO4J_MCP_READ_ONLY", "NEO4J_READ_ONLY"), false),
		Telemetry:                     ParseBool(GetEnvWithAliases("NEO4J_MCP_TELEMETRY", "NEO4J_TELEMETRY"), true),
		LogLevel:                      GetEnvWithAliasesDefault("NEO4J_MCP_LOG_LEVEL", "info", "NEO4J_LOG_LEVEL"),
		LogFormat:                     GetEnvWithAliasesDefault("NEO4J_MCP_LOG_FORMAT", "text", "NEO4J_LOG_FORMAT"),
		SchemaSampleSize:              ParseInt32(GetEnvWithAliases("NEO4J_MCP_SCHEMA_SAMPLE_SIZE", "NEO4J_SCHEMA_SAMPLE_SIZE"), DefaultSchemaSampleSize),
		TransportMode:                 TransportMode(GetEnvWithAliasesDefault("NEO4J_MCP_TRANSPORT_MODE", string(TransportModeStdio), "NEO4J_TRANSPORT_MODE", "NEO4J_MCP_TRANSPORT")),
		HTTPPort:                      GetEnv("NEO4J_MCP_HTTP_PORT"), // Default set after TLS determination
		HTTPHost:                      GetEnvWithDefault("NEO4J_MCP_HTTP_HOST", "127.0.0.1"),
		HTTPAllowedOrigins:            GetEnv("NEO4J_MCP_HTTP_ALLOWED_ORIGINS"),
		HTTPTLSEnabled:                ParseBool(GetEnv("NEO4J_MCP_HTTP_TLS_ENABLED"), false),
		HTTPTLSCertFile:               GetEnv("NEO4J_MCP_HTTP_TLS_CERT_FILE"),
		HTTPTLSKeyFile:                GetEnv("NEO4J_MCP_HTTP_TLS_KEY_FILE"),
		AuthHeaderName:                GetEnvWithAliasesDefault("NEO4J_MCP_HTTP_AUTH_HEADER_NAME", "Authorization", "NEO4J_HTTP_AUTH_HEADER_NAME"),
		AllowUnauthenticatedPing:      ParseBool(GetEnvWithAliases("NEO4J_MCP_HTTP_ALLOW_UNAUTHENTICATED_PING", "NEO4J_HTTP_ALLOW_UNAUTHENTICATED_PING"), false),
		AllowUnauthenticatedToolsList: ParseBool(GetEnvWithAliases("NEO4J_MCP_HTTP_ALLOW_UNAUTHENTICATED_TOOLS_LIST", "NEO4J_HTTP_ALLOW_UNAUTHENTICATED_TOOLS_LIST"), false),
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
		if cliOverrides.TransportMode != "" {
			cfg.TransportMode = TransportMode(cliOverrides.TransportMode)
		}
		if cliOverrides.Port != "" {
			cfg.HTTPPort = cliOverrides.Port
		}
		if cliOverrides.Host != "" {
			cfg.HTTPHost = cliOverrides.Host
		}
		if cliOverrides.AllowedOrigins != "" {
			cfg.HTTPAllowedOrigins = cliOverrides.AllowedOrigins
		}
		if cliOverrides.TLSEnabled != "" {
			cfg.HTTPTLSEnabled = ParseBool(cliOverrides.TLSEnabled, false)
		}
		if cliOverrides.TLSCertFile != "" {
			cfg.HTTPTLSCertFile = cliOverrides.TLSCertFile
		}
		if cliOverrides.TLSKeyFile != "" {
			cfg.HTTPTLSKeyFile = cliOverrides.TLSKeyFile
		}
		if cliOverrides.AuthHeaderName != "" {
			cfg.AuthHeaderName = cliOverrides.AuthHeaderName
		}
		if cliOverrides.AllowUnauthenticatedPing != "" {
			cfg.AllowUnauthenticatedPing = ParseBool(cliOverrides.AllowUnauthenticatedPing, false)
		}
		if cliOverrides.AllowUnauthenticatedToolsList != "" {
			cfg.AllowUnauthenticatedToolsList = ParseBool(cliOverrides.AllowUnauthenticatedToolsList, false)
		}
	}

	// Set default HTTP port based on TLS configuration if not explicitly provided
	// Default to 443 for HTTPS, 80 for HTTP
	if cfg.HTTPPort == "" {
		if cfg.HTTPTLSEnabled {
			cfg.HTTPPort = "443"
		} else {
			cfg.HTTPPort = "80"
		}
	}

	// Normalize and validate
	headName := strings.TrimSpace(cfg.AuthHeaderName)
	if headName == "" {
		return nil, fmt.Errorf("invalid auth header name: explicitly configured header name cannot be empty; unset NEO4J_MCP_HTTP_AUTH_HEADER_NAME or provide a valid header name")
	}
	// store normalized value
	cfg.AuthHeaderName = headName

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

// GetEnvWithAliases returns the canonical environment variable value, falling
// back to deprecated aliases when the canonical variable is unset or empty.
func GetEnvWithAliases(canonical string, aliases ...string) string {
	if value := os.Getenv(canonical); value != "" {
		return value
	}
	for _, alias := range aliases {
		if value := os.Getenv(alias); value != "" {
			return value
		}
	}
	return ""
}

var deprecatedEnvironmentVariables = []struct {
	alias     string
	canonical string
}{
	{alias: "NEO4J_URI", canonical: "NEO4J_MCP_URI"},
	{alias: "NEO4J_USERNAME", canonical: "NEO4J_MCP_USERNAME"},
	{alias: "NEO4J_PASSWORD", canonical: "NEO4J_MCP_PASSWORD"},
	{alias: "NEO4J_DATABASE", canonical: "NEO4J_MCP_DATABASE"},
	{alias: "NEO4J_READ_ONLY", canonical: "NEO4J_MCP_READ_ONLY"},
	{alias: "NEO4J_TELEMETRY", canonical: "NEO4J_MCP_TELEMETRY"},
	{alias: "NEO4J_LOG_LEVEL", canonical: "NEO4J_MCP_LOG_LEVEL"},
	{alias: "NEO4J_LOG_FORMAT", canonical: "NEO4J_MCP_LOG_FORMAT"},
	{alias: "NEO4J_TRANSPORT_MODE", canonical: "NEO4J_MCP_TRANSPORT_MODE"},
	{alias: "NEO4J_MCP_TRANSPORT", canonical: "NEO4J_MCP_TRANSPORT_MODE"},
	{alias: "NEO4J_SCHEMA_SAMPLE_SIZE", canonical: "NEO4J_MCP_SCHEMA_SAMPLE_SIZE"},
	{alias: "NEO4J_HTTP_AUTH_HEADER_NAME", canonical: "NEO4J_MCP_HTTP_AUTH_HEADER_NAME"},
	{alias: "NEO4J_HTTP_ALLOW_UNAUTHENTICATED_PING", canonical: "NEO4J_MCP_HTTP_ALLOW_UNAUTHENTICATED_PING"},
	{alias: "NEO4J_HTTP_ALLOW_UNAUTHENTICATED_TOOLS_LIST", canonical: "NEO4J_MCP_HTTP_ALLOW_UNAUTHENTICATED_TOOLS_LIST"},
}

func warnOnDeprecatedUsage() {
	for _, variable := range deprecatedEnvironmentVariables {
		if os.Getenv(variable.alias) != "" {
			fmt.Fprintf(os.Stderr, DeprecatedVariableMessage, "environment variable", variable.alias, variable.canonical)
		}
	}
}

// GetEnvWithAliasesDefault returns the canonical or deprecated environment
// variable value, or defaultValue when all names are unset or empty.
func GetEnvWithAliasesDefault(canonical, defaultValue string, aliases ...string) string {
	if value := GetEnvWithAliases(canonical, aliases...); value != "" {
		return value
	}
	return defaultValue
}

// GetEnvWithDefault returns the value of an environment variable or a default value
func GetEnvWithDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// GetTransportModeWithDefault returns the value of an environment variable or a default value
func GetTransportModeWithDefault(key, defaultValue TransportMode) TransportMode {
	if value := os.Getenv(string(key)); value != "" {
		return TransportMode(value)
	}
	return defaultValue
}

// ParseBool parses a string to bool using strconv.ParseBool.
// Returns the default value if the string is empty or invalid.
// Logs a warning if the value is non-empty but invalid.
// Accepts: "1", "t", "T", "true", "True", "TRUE" for true
//
//	"0", "f", "F", "false", "False", "FALSE" for false
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

// ParseInt32 parses a string to int32.
// Returns the default value if the string is empty or invalid.
func ParseInt32(value string, defaultValue int32) int32 {
	if value == "" {
		return defaultValue
	}
	parsed, err := strconv.ParseInt(value, 10, 32)
	if err != nil {
		log.Printf("Warning: Invalid integer value %q, using default: %v", value, defaultValue)
		return defaultValue
	}
	return int32(parsed)
}
