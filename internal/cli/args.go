// Copyright (c) "Neo4j"
// Neo4j Sweden AB [http://neo4j.com]

package cli

import (
	"flag"
	"fmt"
	"os"
	"slices"
	"strings"
)

// osExit is a variable that can be mocked in tests
var osExit = os.Exit

const helpText = `neo4j-mcp - Neo4j Model Context Protocol Server

Usage:
  neo4j-mcp [OPTIONS]

Options:
  -h, --help                          Show this help message
  -v, --version                       Show version information
  --neo4j-uri <URI>                   Neo4j connection URI (overrides environment variable NEO4J_URI)
  --neo4j-username <USERNAME>         Database username (overrides environment variable NEO4J_USERNAME)
  --neo4j-password <PASSWORD>         Database password (overrides environment variable NEO4J_PASSWORD)
  --neo4j-database <DATABASE>         Database name (overrides environment variable NEO4J_DATABASE)
  --neo4j-read-only <BOOLEAN>         Enable read-only mode: true or false (overrides environment variable NEO4J_READ_ONLY)
  --neo4j-telemetry <BOOLEAN>         Enable telemetry: true or false (overrides environment variable NEO4J_TELEMETRY)
  --neo4j-schema-sample-size <INT>    Number of nodes to sample for schema inference (overrides environment variable NEO4J_SCHEMA_SAMPLE_SIZE)
  --neo4j-transport-mode <MODE>       MCP Transport mode (e.g., 'stdio', 'http') (overrides environment variable NEO4J_TRANSPORT_MODE & NEO4J_MCP_TRANSPORT(deprecated))
  --neo4j-http-port <PORT>            HTTP server port (overrides environment variable NEO4J_MCP_HTTP_PORT)
  --neo4j-http-host <HOST>            HTTP server host (overrides environment variable NEO4J_MCP_HTTP_HOST)
  --neo4j-http-allowed-origins <ORIGINS> Comma-separated list of allowed CORS origins (overrides environment variable NEO4J_MCP_HTTP_ALLOWED_ORIGINS)
  --neo4j-http-tls-enabled <BOOLEAN>  Enable TLS/HTTPS for HTTP server: true or false (overrides environment variable NEO4J_MCP_HTTP_TLS_ENABLED)
  --neo4j-http-tls-cert-file <PATH>   Path to TLS certificate file (overrides environment variable NEO4J_MCP_HTTP_TLS_CERT_FILE)
  --neo4j-http-tls-key-file <PATH>    Path to TLS private key file (overrides environment variable NEO4J_MCP_HTTP_TLS_KEY_FILE)
  --neo4j-http-auth-header-name <HEADER> Name of the HTTP header to read auth credentials from (overrides NEO4J_HTTP_AUTH_HEADER_NAME)

Required Environment Variables:
  NEO4J_URI       Neo4j database URI
  NEO4J_USERNAME  Database username
  NEO4J_PASSWORD  Database password

Optional Environment Variables:
  NEO4J_DATABASE  Database name (default: neo4j)
  NEO4J_TELEMETRY Enable/disable telemetry (default: true)
  NEO4J_READ_ONLY Enable read-only mode (default: false)
  NEO4J_SCHEMA_SAMPLE_SIZE Number of nodes to sample for schema inference (default: 100)
  NEO4J_TRANSPORT_MODE MCP Transport mode (e.g., 'stdio', 'http') (default: stdio)
  NEO4J_MCP_TRANSPORT MCP Transport mode (e.g., 'stdio', 'http') (default: stdio)
  NEO4J_MCP_HTTP_PORT HTTP server port (default: 443 with TLS, 80 without TLS)
  NEO4J_MCP_HTTP_HOST HTTP server host (default: 127.0.0.1)
  NEO4J_MCP_HTTP_ALLOWED_ORIGINS Comma-separated list of allowed CORS origins (optional)
  NEO4J_MCP_HTTP_TLS_ENABLED Enable TLS/HTTPS for HTTP server (default: false)
  NEO4J_MCP_HTTP_TLS_CERT_FILE Path to TLS certificate file (required when TLS is enabled)
  NEO4J_MCP_HTTP_TLS_KEY_FILE Path to TLS private key file (required when TLS is enabled)
  NEO4J_HTTP_AUTH_HEADER_NAME Name of the HTTP header to read auth credentials from (default: Authorization)

Examples:
  # Using environment variables
  NEO4J_URI=bolt://localhost:7687 NEO4J_USERNAME=neo4j NEO4J_PASSWORD=password neo4j-mcp

  # Using CLI flags (takes precedence over environment variables)
  neo4j-mcp --neo4j-uri bolt://localhost:7687 --neo4j-username neo4j --neo4j-password password

For more information, visit: https://github.com/neo4j/mcp
`

// Args holds configuration values parsed from command-line flags
type Args struct {
	URI                string
	Username           string
	Password           string
	Database           string
	ReadOnly           string
	Telemetry          string
	SchemaSampleSize   string
	TransportMode      string
	HTTPPort           string
	HTTPHost           string
	HTTPAllowedOrigins string
	HTTPTLSEnabled     string
	HTTPTLSCertFile    string
	HTTPTLSKeyFile     string
	AuthHeaderName     string
}

// this is a list of known configuration flags to be skipped in HandleArgs
// add new config flags here as needed
var argsSlice = []string{
	"--neo4j-uri",
	"--neo4j-username",
	"--neo4j-password",
	"--neo4j-database",
	"--neo4j-read-only",
	"--neo4j-telemetry",
	"--neo4j-schema-sample-size",
	"--neo4j-transport-mode",
	"--neo4j-http-port",
	"--neo4j-http-host",
	"--neo4j-http-allowed-origins",
	"--neo4j-http-tls-enabled",
	"--neo4j-http-tls-cert-file",
	"--neo4j-http-tls-key-file",
	"--neo4j-auth-header-name",
}

// ParseConfigFlags parses CLI flags and returns configuration values.
// It should be called after HandleArgs to ensure help/version flags are processed first.
func ParseConfigFlags() *Args {
	neo4jURI := flag.String("neo4j-uri", "", "Neo4j connection URI (overrides NEO4J_URI env var)")
	neo4jUsername := flag.String("neo4j-username", "", "Neo4j username (overrides NEO4J_USERNAME env var)")
	neo4jPassword := flag.String("neo4j-password", "", "Neo4j password (overrides NEO4J_PASSWORD env var)")
	neo4jDatabase := flag.String("neo4j-database", "", "Neo4j database name (overrides NEO4J_DATABASE env var)")
	neo4jReadOnly := flag.String("neo4j-read-only", "", "Enable read-only mode: true or false (overrides NEO4J_READ_ONLY env var)")
	neo4jTelemetry := flag.String("neo4j-telemetry", "", "Enable telemetry: true or false (overrides NEO4J_TELEMETRY env var)")
	neo4jSchemaSampleSize := flag.String("neo4j-schema-sample-size", "", "Number of nodes to sample for schema inference (overrides NEO4J_SCHEMA_SAMPLE_SIZE env var)")
	neo4jTransportMode := flag.String("neo4j-transport-mode", "", "MCP Transport mode (e.g., 'stdio', 'http') (overrides NEO4J_TRANSPORT_MODE env var)")
	neo4jHTTPPort := flag.String("neo4j-http-port", "", "HTTP server port (overrides NEO4J_MCP_HTTP_PORT env var)")
	neo4jHTTPHost := flag.String("neo4j-http-host", "", "HTTP server host (overrides NEO4J_MCP_HTTP_HOST env var)")
	neo4jHTTPAllowedOrigins := flag.String("neo4j-http-allowed-origins", "", "Comma-separated list of allowed CORS origins (overrides NEO4J_MCP_HTTP_ALLOWED_ORIGINS env var)")
	neo4jHTTPTLSEnabled := flag.String("neo4j-http-tls-enabled", "", "Enable TLS/HTTPS for HTTP server: true or false (overrides NEO4J_MCP_HTTP_TLS_ENABLED env var)")
	neo4jHTTPTLSCertFile := flag.String("neo4j-http-tls-cert-file", "", "Path to TLS certificate file (overrides NEO4J_MCP_HTTP_TLS_CERT_FILE env var)")
	neo4jHTTPTLSKeyFile := flag.String("neo4j-http-tls-key-file", "", "Path to TLS private key file (overrides NEO4J_MCP_HTTP_TLS_KEY_FILE env var)")
	neo4jAuthHeaderName := flag.String("neo4j-http-auth-header-name", "", "Name of the HTTP header to read auth credentials from (overrides NEO4J_HTTP_AUTH_HEADER_NAME env var)")

	flag.Parse()

	return &Args{
		URI:                *neo4jURI,
		Username:           *neo4jUsername,
		Password:           *neo4jPassword,
		Database:           *neo4jDatabase,
		ReadOnly:           *neo4jReadOnly,
		Telemetry:          *neo4jTelemetry,
		SchemaSampleSize:   *neo4jSchemaSampleSize,
		TransportMode:      *neo4jTransportMode,
		HTTPPort:           *neo4jHTTPPort,
		HTTPHost:           *neo4jHTTPHost,
		HTTPAllowedOrigins: *neo4jHTTPAllowedOrigins,
		HTTPTLSEnabled:     *neo4jHTTPTLSEnabled,
		HTTPTLSCertFile:    *neo4jHTTPTLSCertFile,
		HTTPTLSKeyFile:     *neo4jHTTPTLSKeyFile,
		AuthHeaderName:     *neo4jAuthHeaderName,
	}
}

// HandleArgs processes command-line arguments for version and help flags.
// It exits the program after displaying the requested information.
// If unknown flags are encountered, it prints an error message and exits.
// Known configuration flags are skipped here so that the flag package in main.go can handle them properly.
func HandleArgs(version string) {
	if len(os.Args) <= 1 {
		return
	}

	flags := make(map[string]bool)
	var err error
	i := 1 // we start from 1 because os.Args[0] is the program name ("neo4j-mcp") - not a flag

	for i < len(os.Args) {
		arg := os.Args[i]

		// Allow configuration flags to be parsed by the flag package
		if slices.Contains(argsSlice, arg) {
			// Check if there's a value following the flag
			if i+1 >= len(os.Args) {
				err = fmt.Errorf("%s requires a value", arg)
				break
			}
			// Check if next argument is another flag (starts with -)
			nextArg := os.Args[i+1]
			if strings.HasPrefix(nextArg, "-") {
				err = fmt.Errorf("%s requires a value (got flag %s instead)", arg, nextArg)
				break
			}
			// Safe to skip flag and value - let flag package handle them
			i += 2
			continue
		}

		switch arg {
		case "-h", "--help":
			flags["help"] = true
			i++
		case "-v", "--version":
			flags["version"] = true
			i++
		default:
			if arg == "--" {
				// Stop processing our flags, let flag package handle the rest
				i = len(os.Args)
			} else {
				err = fmt.Errorf("unknown flag or argument: %s", arg)
				i++
			}
		}
		// Exit loop if an error occurred
		if err != nil {
			break
		}
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		osExit(1)
	}

	if flags["help"] {
		fmt.Print(helpText)
		osExit(0)
	}

	if flags["version"] {
		fmt.Printf("neo4j-mcp version: %s\n", version)
		osExit(0)
	}
}
