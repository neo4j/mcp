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
  --uri <URI>                         Neo4j connection URI (overrides NEO4J_MCP_URI)
  --username <USERNAME>               Database username (overrides NEO4J_MCP_USERNAME)
  --password <PASSWORD>               Database password (overrides NEO4J_MCP_PASSWORD)
  --database <DATABASE>               Database name (overrides NEO4J_MCP_DATABASE)
  --read-only <BOOLEAN>               Enable read-only mode: true or false (overrides NEO4J_MCP_READ_ONLY)
  --tools <TOOLS>                     Define tools available by filtering tools returned in tools/list response
  --telemetry <BOOLEAN>               Enable telemetry: true or false (overrides NEO4J_MCP_TELEMETRY)
  --schema-sample-size <INT>          Number of nodes to sample for schema inference (overrides NEO4J_MCP_SCHEMA_SAMPLE_SIZE)
  --transport-mode <MODE>             MCP transport mode: 'stdio' or 'http' (overrides NEO4J_MCP_TRANSPORT_MODE)
  --http-port <PORT>                  HTTP server port (overrides NEO4J_MCP_HTTP_PORT)
  --http-host <HOST>                  HTTP server host (overrides NEO4J_MCP_HTTP_HOST)
  --http-allowed-origins <ORIGINS>    Comma-separated list of allowed CORS origins (overrides NEO4J_MCP_HTTP_ALLOWED_ORIGINS)
  --http-tls-enabled <BOOLEAN>        Enable TLS/HTTPS for HTTP server: true or false (overrides NEO4J_MCP_HTTP_TLS_ENABLED)
  --http-tls-cert-file <PATH>         Path to TLS certificate file (overrides NEO4J_MCP_HTTP_TLS_CERT_FILE)
  --http-tls-key-file <PATH>          Path to TLS private key file (overrides NEO4J_MCP_HTTP_TLS_KEY_FILE)
  --http-auth-header-name <HEADER>    Name of the HTTP header to read auth credentials from (overrides NEO4J_MCP_HTTP_AUTH_HEADER_NAME)
  --http-allow-unauthenticated-ping <BOOLEAN> Allow unauthenticated ping (overrides NEO4J_MCP_HTTP_ALLOW_UNAUTHENTICATED_PING)
  --http-allow-unauthenticated-tools-list <BOOLEAN> Allow unauthenticated tools/list (overrides NEO4J_MCP_HTTP_ALLOW_UNAUTHENTICATED_TOOLS_LIST)
  --request-timeout <DURATION>       Maximum duration for a single MCP request, up to 30m. Also caps the per-request X-Neo4j-MCP-Request-Timeout header in HTTP mode (overrides environment variable NEO4J_MCP_REQUEST_TIMEOUT)

Required Environment Variables (STDIO mode):
  NEO4J_MCP_URI       Neo4j database URI
  NEO4J_MCP_USERNAME  Database username
  NEO4J_MCP_PASSWORD  Database password

Optional Environment Variables:
  NEO4J_MCP_DATABASE  Database name (default: neo4j)
  NEO4J_MCP_TELEMETRY Enable/disable telemetry (default: true)
  NEO4J_MCP_READ_ONLY Enable read-only mode (default: false)
  NEO4J_MCP_SCHEMA_SAMPLE_SIZE Number of nodes to sample for schema inference (default: 100)
  NEO4J_MCP_LOG_LEVEL Log level (default: info)
  NEO4J_MCP_LOG_FORMAT Log format: text or json (default: text)
  NEO4J_MCP_TOOLS Define tools available by filtering tools returned in tools/list response (default: all tools enabled)
  NEO4J_MCP_TRANSPORT_MODE MCP transport mode (default: stdio)
  NEO4J_MCP_HTTP_PORT HTTP server port (default: 443 with TLS, 80 without TLS)
  NEO4J_MCP_HTTP_HOST HTTP server host (default: 127.0.0.1)
  NEO4J_MCP_HTTP_ALLOWED_ORIGINS Comma-separated list of allowed CORS origins (optional)
  NEO4J_MCP_HTTP_TLS_ENABLED Enable TLS/HTTPS for HTTP server (default: false)
  NEO4J_MCP_HTTP_TLS_CERT_FILE Path to TLS certificate file (required when TLS is enabled)
  NEO4J_MCP_HTTP_TLS_KEY_FILE Path to TLS private key file (required when TLS is enabled)
  NEO4J_MCP_HTTP_AUTH_HEADER_NAME Name of the HTTP header to read auth credentials from (default: Authorization)
  NEO4J_MCP_HTTP_ALLOW_UNAUTHENTICATED_PING Allow unauthenticated ping health checks (default: false)
  NEO4J_MCP_HTTP_ALLOW_UNAUTHENTICATED_TOOLS_LIST Allow unauthenticated tool listing (default: false)
  NEO4J_MCP_REQUEST_TIMEOUT Maximum duration for a single MCP request (default: 3m, maximum: 30m)

Examples:
  # Using environment variables
  NEO4J_MCP_URI=bolt://localhost:7687 NEO4J_MCP_USERNAME=neo4j NEO4J_MCP_PASSWORD=password NEO4J_MCP_DATABASE=neo4j neo4j-mcp

  # Using CLI flags (takes precedence over environment variables)
  neo4j-mcp --uri bolt://localhost:7687 --username neo4j --password password --database neo4j

For more information, visit: https://github.com/neo4j/mcp
`

// Args holds configuration values parsed from command-line flags
type Args struct {
	URI                               string
	Username                          string
	Password                          string // #nosec G117 -- Password is only used during startup to create auth token, not logged or exposed
	Database                          string
	ReadOnly                          string
	Tools                             *string // allows explicit empty string arguments to be differentiated from unset arguments
	Telemetry                         string
	SchemaSampleSize                  string
	TransportMode                     string
	HTTPPort                          string
	HTTPHost                          string
	HTTPAllowedOrigins                string
	HTTPTLSEnabled                    string
	HTTPTLSCertFile                   string
	HTTPTLSKeyFile                    string
	AuthHeaderName                    string
	HTTPAllowUnauthenticatedPing      string
	HTTPAllowUnauthenticatedToolsList string
	RequestTimeout                    string
}

// this is a list of known configuration flags to be skipped in HandleArgs
// add new config flags here as needed
var argsSlice = []string{
	"--uri",
	"--username",
	"--password",
	"--database",
	"--read-only",
	"--tools",
	"--telemetry",
	"--schema-sample-size",
	"--transport-mode",
	"--http-port",
	"--http-host",
	"--http-allowed-origins",
	"--http-tls-enabled",
	"--http-tls-cert-file",
	"--http-tls-key-file",
	"--http-auth-header-name",
	"--http-allow-unauthenticated-ping",
	"--http-allow-unauthenticated-tools-list",
	"--request-timeout",
}

// ParseConfigFlags parses CLI flags and returns configuration values.
// It should be called after HandleArgs to ensure help/version flags are processed first.
func ParseConfigFlags() *Args {
	uri := flag.String("uri", "", "Neo4j connection URI (overrides NEO4J_MCP_URI env var)")
	username := flag.String("username", "", "Neo4j username (overrides NEO4J_MCP_USERNAME env var)")
	password := flag.String("password", "", "Neo4j password (overrides NEO4J_MCP_PASSWORD env var)")
	database := flag.String("database", "", "Neo4j database name (overrides NEO4J_MCP_DATABASE env var)")
	readOnly := flag.String("read-only", "", "Enable read-only mode: true or false (overrides NEO4J_MCP_READ_ONLY env var)")
	var tools *string
	flag.Func("tools", "Define tools available by filtering tools returned in tools/list response", func(s string) error {
		if s == "" {
			return fmt.Errorf("cannot be empty; omit the flag to use all tools, or provide a comma-separated list of tools")
		}
		tools = &s
		return nil
	})
	telemetry := flag.String("telemetry", "", "Enable telemetry: true or false (overrides NEO4J_MCP_TELEMETRY env var)")
	schemaSampleSize := flag.String("schema-sample-size", "", "Number of nodes to sample for schema inference (overrides NEO4J_MCP_SCHEMA_SAMPLE_SIZE env var)")
	transportMode := flag.String("transport-mode", "", "MCP transport mode: stdio or http (overrides NEO4J_MCP_TRANSPORT_MODE env var)")
	httpPort := flag.String("http-port", "", "HTTP server port (overrides NEO4J_MCP_HTTP_PORT env var)")
	httpHost := flag.String("http-host", "", "HTTP server host (overrides NEO4J_MCP_HTTP_HOST env var)")
	httpAllowedOrigins := flag.String("http-allowed-origins", "", "Comma-separated list of allowed CORS origins (overrides NEO4J_MCP_HTTP_ALLOWED_ORIGINS env var)")
	httpTLSEnabled := flag.String("http-tls-enabled", "", "Enable TLS/HTTPS for HTTP server: true or false (overrides NEO4J_MCP_HTTP_TLS_ENABLED env var)")
	httpTLSCertFile := flag.String("http-tls-cert-file", "", "Path to TLS certificate file (overrides NEO4J_MCP_HTTP_TLS_CERT_FILE env var)")
	httpTLSKeyFile := flag.String("http-tls-key-file", "", "Path to TLS private key file (overrides NEO4J_MCP_HTTP_TLS_KEY_FILE env var)")
	authHeaderName := flag.String("http-auth-header-name", "", "Name of the HTTP header to read auth credentials from (overrides NEO4J_MCP_HTTP_AUTH_HEADER_NAME env var)")
	allowUnauthenticatedPing := flag.String("http-allow-unauthenticated-ping", "", "Allow unauthenticated ping: true or false (overrides NEO4J_MCP_HTTP_ALLOW_UNAUTHENTICATED_PING env var)")
	allowUnauthenticatedToolsList := flag.String("http-allow-unauthenticated-tools-list", "", "Allow unauthenticated tools/list: true or false (overrides NEO4J_MCP_HTTP_ALLOW_UNAUTHENTICATED_TOOLS_LIST env var)")
	requestTimeout := flag.String("request-timeout", "", "Maximum duration for a single MCP request, up to 30m (overrides NEO4J_MCP_REQUEST_TIMEOUT env var)")
	flag.Parse()

	return &Args{
		URI:                               *uri,
		Username:                          *username,
		Password:                          *password,
		Database:                          *database,
		ReadOnly:                          *readOnly,
		Tools:                             tools,
		Telemetry:                         *telemetry,
		SchemaSampleSize:                  *schemaSampleSize,
		TransportMode:                     *transportMode,
		HTTPPort:                          *httpPort,
		HTTPHost:                          *httpHost,
		HTTPAllowedOrigins:                *httpAllowedOrigins,
		HTTPTLSEnabled:                    *httpTLSEnabled,
		HTTPTLSCertFile:                   *httpTLSCertFile,
		HTTPTLSKeyFile:                    *httpTLSKeyFile,
		HTTPAllowUnauthenticatedPing:      *allowUnauthenticatedPing,
		HTTPAllowUnauthenticatedToolsList: *allowUnauthenticatedToolsList,
		AuthHeaderName:                    *authHeaderName,
		RequestTimeout:                    *requestTimeout,
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
