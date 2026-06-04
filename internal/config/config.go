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

	"github.com/neo4j/mcp/internal/logger"
)

type TransportMode string

const (
	// DefaultSchemaSampleSize is the default number of nodes to sample per label when inferring schema
	DefaultSchemaSampleSize   int32         = 100
	TransportModeStdio        TransportMode = "stdio"
	TransportModeHTTP         TransportMode = "http"
	DeprecatedVariableMessage string        = "Warning: deprecated environment variable \"%s\". Please use: \"%s\" instead\n"
)

// ValidTransportModes defines the allowed transport mode values
var ValidTransportModes = []TransportMode{TransportModeStdio, TransportModeHTTP}

// AvailableTools defines the available MCP tools
var AvailableTools = []string{"read-cypher", "write-cypher", "list-gds-procedures", "get-schema"}

// Config holds the application configuration
type Config struct {
	URI                           string
	Username                      string
	Password                      string // #nosec G117
	Database                      string
	ReadOnly                      bool // If true, disables write tools
	Tools                         []string
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

	// Default to stdio if not provided (maintains backward compatibility with tests constructing Config directly)
	if c.TransportMode == "" {
		c.TransportMode = TransportModeStdio
	}

	// Validate transport mode
	if !slices.Contains(ValidTransportModes, c.TransportMode) {
		return fmt.Errorf("invalid transport mode '%s', must be one of %v", c.TransportMode, ValidTransportModes)
	}

	// For STDIO mode, require URI, username, password, and database from environment.
	// For HTTP mode, URI, credentials and database come per-request (X-Neo4j-MCP-URI header, Auth headers, and URL path);
	if c.TransportMode == TransportModeStdio {
		if c.URI == "" {
			return fmt.Errorf("Neo4j URI is required for STDIO mode but was empty")
		}
		if c.Username == "" {
			return fmt.Errorf("Neo4j username is required for STDIO mode")
		}
		if c.Password == "" {
			return fmt.Errorf("Neo4j password is required for STDIO mode")
		}
		if c.Database == "" {
			return fmt.Errorf("Neo4j database is required for STDIO mode (set NEO4J_DATABASE or use --neo4j-database flag)")
		}
	} else {
		if c.URI != "" {
			return fmt.Errorf("Neo4j URI should not be set for HTTP transport mode; URI is provided per-request via X-Neo4j-MCP-URI header")
		}
		if c.Username != "" || c.Password != "" {
			return fmt.Errorf("Neo4j username and password should not be set for HTTP transport mode; credentials are provided per-request via Auth headers")
		}
		if c.Database != "" {
			return fmt.Errorf("NEO4J_DATABASE environment variable or --neo4j-database flag should not be set for HTTP transport mode; database is selected per-request via URL path (e.g., /db/{databaseName}/mcp)")
		}
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

	for _, toolName := range c.Tools {
		if !slices.Contains(AvailableTools, toolName) {
			return fmt.Errorf("tool %q is invalid. Available tools are: %s", toolName, strings.Join(AvailableTools, ", "))
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
	Tools                         *string
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
	logLevel := GetEnvWithDefault("NEO4J_LOG_LEVEL", "info")
	logFormat := GetEnvWithDefault("NEO4J_LOG_FORMAT", "text")

	// Validate log level and use default if invalid
	if !slices.Contains(logger.ValidLogLevels, logLevel) {
		fmt.Fprintf(os.Stderr, "Warning: invalid NEO4J_LOG_LEVEL '%s', using default 'info'. Valid values: %v\n", logLevel, logger.ValidLogLevels)
		logLevel = "info"
	}

	// Validate log format and use default if invalid
	if !slices.Contains(logger.ValidLogFormats, logFormat) {
		fmt.Fprintf(os.Stderr, "Warning: invalid NEO4J_LOG_FORMAT '%s', using default 'text'. Valid values: %v\n", logFormat, logger.ValidLogFormats)
		logFormat = "text"
	}

	if GetEnv("NEO4J_MCP_TRANSPORT") != "" {
		fmt.Fprintf(os.Stderr, DeprecatedVariableMessage, "NEO4J_MCP_TRANSPORT", "NEO4J_TRANSPORT_MODE")
	}

	cfg := &Config{
		URI:                           GetEnv("NEO4J_URI"),
		Username:                      GetEnv("NEO4J_USERNAME"),
		Password:                      GetEnv("NEO4J_PASSWORD"),
		Database:                      GetEnv("NEO4J_DATABASE"),
		ReadOnly:                      ParseBool(GetEnv("NEO4J_READ_ONLY"), false),
		Telemetry:                     ParseBool(GetEnv("NEO4J_TELEMETRY"), true),
		LogLevel:                      logLevel,
		LogFormat:                     logFormat,
		SchemaSampleSize:              ParseInt32(GetEnv("NEO4J_SCHEMA_SAMPLE_SIZE"), DefaultSchemaSampleSize),
		TransportMode:                 GetTransportModeWithDefault("NEO4J_TRANSPORT_MODE", GetTransportModeWithDefault("NEO4J_MCP_TRANSPORT", TransportModeStdio)),
		HTTPPort:                      GetEnv("NEO4J_MCP_HTTP_PORT"), // Default set after TLS determination
		HTTPHost:                      GetEnvWithDefault("NEO4J_MCP_HTTP_HOST", "127.0.0.1"),
		HTTPAllowedOrigins:            GetEnv("NEO4J_MCP_HTTP_ALLOWED_ORIGINS"),
		HTTPTLSEnabled:                ParseBool(GetEnv("NEO4J_MCP_HTTP_TLS_ENABLED"), false),
		HTTPTLSCertFile:               GetEnv("NEO4J_MCP_HTTP_TLS_CERT_FILE"),
		HTTPTLSKeyFile:                GetEnv("NEO4J_MCP_HTTP_TLS_KEY_FILE"),
		AuthHeaderName:                GetEnvWithDefault("NEO4J_HTTP_AUTH_HEADER_NAME", "Authorization"),
		AllowUnauthenticatedPing:      ParseBool(GetEnv("NEO4J_HTTP_ALLOW_UNAUTHENTICATED_PING"), false),
		AllowUnauthenticatedToolsList: ParseBool(GetEnv("NEO4J_HTTP_ALLOW_UNAUTHENTICATED_TOOLS_LIST"), false),
	}

	if toolsEnv, ok := os.LookupEnv("NEO4J_MCP_TOOLS"); ok {
		if toolsEnv == "" {
			return nil, fmt.Errorf("invalid tools configuration: NEO4J_MCP_TOOLS is set but empty; provide a comma-separated list of tools or unset the variable")
		}
		cfg.Tools = ParseCommaSeparatedString(toolsEnv)
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
		if cliOverrides.Tools != nil {
			cfg.Tools = ParseCommaSeparatedString(*cliOverrides.Tools)
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

	// If tools haven't been set at this point, they have neither been provided nor explicitly unset
	// Default to all available tools
	if cfg.Tools == nil {
		cfg.Tools = AvailableTools
	}

	// Normalize and validate
	headName := strings.TrimSpace(cfg.AuthHeaderName)
	if headName == "" {
		return nil, fmt.Errorf("invalid auth header name: explicitly configured header name cannot be empty; unset NEO4J_HTTP_AUTH_HEADER_NAME or provide a valid header name")
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

// ParseCommaSeparatedString parses a comma-separated string into a slice of strings.
// Ensures that whitespace, trailing and leading commas are ignored.
func ParseCommaSeparatedString(value string) []string {
	parts := strings.Split(value, ",")
	n := 0
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			parts[n] = p
			n++
		}
	}
	return parts[:n]
}
