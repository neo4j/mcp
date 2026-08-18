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
  --telemetry <BOOLEAN>               Enable telemetry: true or false (overrides NEO4J_MCP_TELEMETRY)
  --schema-sample-size <INT>          Number of nodes to sample for schema inference (overrides NEO4J_MCP_SCHEMA_SAMPLE_SIZE)
  --transport <MODE>                  MCP transport mode: 'stdio' or 'http' (overrides NEO4J_MCP_TRANSPORT_MODE)
  --http-port <PORT>                  HTTP server port (overrides NEO4J_MCP_HTTP_PORT)
  --http-host <HOST>                  HTTP server host (overrides NEO4J_MCP_HTTP_HOST)
  --http-allowed-origins <ORIGINS>    Comma-separated list of allowed CORS origins (overrides NEO4J_MCP_HTTP_ALLOWED_ORIGINS)
  --http-tls-enabled <BOOLEAN>        Enable TLS/HTTPS for HTTP server: true or false (overrides NEO4J_MCP_HTTP_TLS_ENABLED)
  --http-tls-cert-file <PATH>         Path to TLS certificate file (overrides NEO4J_MCP_HTTP_TLS_CERT_FILE)
  --http-tls-key-file <PATH>          Path to TLS private key file (overrides NEO4J_MCP_HTTP_TLS_KEY_FILE)
  --http-auth-header-name <HEADER>    Name of the HTTP header to read auth credentials from (overrides NEO4J_MCP_HTTP_AUTH_HEADER_NAME)
  --http-allow-unauthenticated-ping <BOOLEAN> Allow unauthenticated ping (overrides NEO4J_MCP_HTTP_ALLOW_UNAUTHENTICATED_PING)
  --http-allow-unauthenticated-tools-list <BOOLEAN> Allow unauthenticated tools/list (overrides NEO4J_MCP_HTTP_ALLOW_UNAUTHENTICATED_TOOLS_LIST)

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

Deprecated environment variables and --neo4j-* flags remain accepted in v1 and emit a warning. They will be removed in v2. The deprecated environment aliases are the previous unscoped names shown in the project changelog.

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
}

// this is a list of known configuration flags to be skipped in HandleArgs
// add new config flags here as needed
var argsSlice = []string{
	"--uri",
	"--neo4j-uri",
	"--username",
	"--neo4j-username",
	"--password",
	"--neo4j-password",
	"--database",
	"--neo4j-database",
	"--read-only",
	"--neo4j-read-only",
	"--telemetry",
	"--neo4j-telemetry",
	"--schema-sample-size",
	"--neo4j-schema-sample-size",
	"--transport",
	"--neo4j-transport-mode",
	"--http-port",
	"--neo4j-http-port",
	"--http-host",
	"--neo4j-http-host",
	"--http-allowed-origins",
	"--neo4j-http-allowed-origins",
	"--http-tls-enabled",
	"--neo4j-http-tls-enabled",
	"--http-tls-cert-file",
	"--neo4j-http-tls-cert-file",
	"--http-tls-key-file",
	"--neo4j-http-tls-key-file",
	"--http-auth-header-name",
	"--neo4j-http-auth-header-name",
	"--http-allow-unauthenticated-ping",
	"--neo4j-http-allow-unauthenticated-ping",
	"--http-allow-unauthenticated-tools-list",
	"--neo4j-http-allow-unauthenticated-tools-list",
}

// ParseConfigFlags parses CLI flags and returns configuration values.
// It should be called after HandleArgs to ensure help/version flags are processed first.
const deprecatedFlagMessage = "Warning: deprecated CLI flag %q; use %q instead. Support will be removed in v2.\n"

func mergeFlagValue(canonical, deprecated *string, canonicalName, deprecatedName string) string {
	if *deprecated != "" {
		fmt.Fprintf(os.Stderr, deprecatedFlagMessage, deprecatedName, canonicalName)
	}
	if *canonical != "" {
		return *canonical
	}
	return *deprecated
}

func ParseConfigFlags() *Args {
	uri := flag.String("uri", "", "Neo4j connection URI (overrides NEO4J_MCP_URI env var)")
	neo4jURI := flag.String("neo4j-uri", "", "Deprecated alias for --uri")
	username := flag.String("username", "", "Neo4j username (overrides NEO4J_MCP_USERNAME env var)")
	neo4jUsername := flag.String("neo4j-username", "", "Deprecated alias for --username")
	password := flag.String("password", "", "Neo4j password (overrides NEO4J_MCP_PASSWORD env var)")
	neo4jPassword := flag.String("neo4j-password", "", "Deprecated alias for --password")
	database := flag.String("database", "", "Neo4j database name (overrides NEO4J_MCP_DATABASE env var)")
	neo4jDatabase := flag.String("neo4j-database", "", "Deprecated alias for --database")
	readOnly := flag.String("read-only", "", "Enable read-only mode: true or false (overrides NEO4J_MCP_READ_ONLY env var)")
	neo4jReadOnly := flag.String("neo4j-read-only", "", "Deprecated alias for --read-only")
	telemetry := flag.String("telemetry", "", "Enable telemetry: true or false (overrides NEO4J_MCP_TELEMETRY env var)")
	neo4jTelemetry := flag.String("neo4j-telemetry", "", "Deprecated alias for --telemetry")
	schemaSampleSize := flag.String("schema-sample-size", "", "Number of nodes to sample for schema inference (overrides NEO4J_MCP_SCHEMA_SAMPLE_SIZE env var)")
	neo4jSchemaSampleSize := flag.String("neo4j-schema-sample-size", "", "Deprecated alias for --schema-sample-size")
	transport := flag.String("transport", "", "MCP transport mode: stdio or http (overrides NEO4J_MCP_TRANSPORT_MODE env var)")
	neo4jTransportMode := flag.String("neo4j-transport-mode", "", "Deprecated alias for --transport")
	httpPort := flag.String("http-port", "", "HTTP server port (overrides NEO4J_MCP_HTTP_PORT env var)")
	neo4jHTTPPort := flag.String("neo4j-http-port", "", "Deprecated alias for --http-port")
	httpHost := flag.String("http-host", "", "HTTP server host (overrides NEO4J_MCP_HTTP_HOST env var)")
	neo4jHTTPHost := flag.String("neo4j-http-host", "", "Deprecated alias for --http-host")
	httpAllowedOrigins := flag.String("http-allowed-origins", "", "Comma-separated list of allowed CORS origins (overrides NEO4J_MCP_HTTP_ALLOWED_ORIGINS env var)")
	neo4jHTTPAllowedOrigins := flag.String("neo4j-http-allowed-origins", "", "Deprecated alias for --http-allowed-origins")
	httpTLSEnabled := flag.String("http-tls-enabled", "", "Enable TLS/HTTPS for HTTP server: true or false (overrides NEO4J_MCP_HTTP_TLS_ENABLED env var)")
	neo4jHTTPTLSEnabled := flag.String("neo4j-http-tls-enabled", "", "Deprecated alias for --http-tls-enabled")
	httpTLSCertFile := flag.String("http-tls-cert-file", "", "Path to TLS certificate file (overrides NEO4J_MCP_HTTP_TLS_CERT_FILE env var)")
	neo4jHTTPTLSCertFile := flag.String("neo4j-http-tls-cert-file", "", "Deprecated alias for --http-tls-cert-file")
	httpTLSKeyFile := flag.String("http-tls-key-file", "", "Path to TLS private key file (overrides NEO4J_MCP_HTTP_TLS_KEY_FILE env var)")
	neo4jHTTPTLSKeyFile := flag.String("neo4j-http-tls-key-file", "", "Deprecated alias for --http-tls-key-file")
	authHeaderName := flag.String("http-auth-header-name", "", "Name of the HTTP header to read auth credentials from (overrides NEO4J_MCP_HTTP_AUTH_HEADER_NAME env var)")
	neo4jAuthHeaderName := flag.String("neo4j-http-auth-header-name", "", "Deprecated alias for --http-auth-header-name")
	allowUnauthenticatedPing := flag.String("http-allow-unauthenticated-ping", "", "Allow unauthenticated ping: true or false (overrides NEO4J_MCP_HTTP_ALLOW_UNAUTHENTICATED_PING env var)")
	neo4jHTTPAllowUnauthenticatedPing := flag.String("neo4j-http-allow-unauthenticated-ping", "", "Deprecated alias for --http-allow-unauthenticated-ping")
	allowUnauthenticatedToolsList := flag.String("http-allow-unauthenticated-tools-list", "", "Allow unauthenticated tools/list: true or false (overrides NEO4J_MCP_HTTP_ALLOW_UNAUTHENTICATED_TOOLS_LIST env var)")
	neo4jHTTPAllowUnauthenticatedToolsList := flag.String("neo4j-http-allow-unauthenticated-tools-list", "", "Deprecated alias for --http-allow-unauthenticated-tools-list")

	flag.Parse()

	return &Args{
		URI:                               mergeFlagValue(uri, neo4jURI, "--uri", "--neo4j-uri"),
		Username:                          mergeFlagValue(username, neo4jUsername, "--username", "--neo4j-username"),
		Password:                          mergeFlagValue(password, neo4jPassword, "--password", "--neo4j-password"),
		Database:                          mergeFlagValue(database, neo4jDatabase, "--database", "--neo4j-database"),
		ReadOnly:                          mergeFlagValue(readOnly, neo4jReadOnly, "--read-only", "--neo4j-read-only"),
		Telemetry:                         mergeFlagValue(telemetry, neo4jTelemetry, "--telemetry", "--neo4j-telemetry"),
		SchemaSampleSize:                  mergeFlagValue(schemaSampleSize, neo4jSchemaSampleSize, "--schema-sample-size", "--neo4j-schema-sample-size"),
		TransportMode:                     mergeFlagValue(transport, neo4jTransportMode, "--transport", "--neo4j-transport-mode"),
		HTTPPort:                          mergeFlagValue(httpPort, neo4jHTTPPort, "--http-port", "--neo4j-http-port"),
		HTTPHost:                          mergeFlagValue(httpHost, neo4jHTTPHost, "--http-host", "--neo4j-http-host"),
		HTTPAllowedOrigins:                mergeFlagValue(httpAllowedOrigins, neo4jHTTPAllowedOrigins, "--http-allowed-origins", "--neo4j-http-allowed-origins"),
		HTTPTLSEnabled:                    mergeFlagValue(httpTLSEnabled, neo4jHTTPTLSEnabled, "--http-tls-enabled", "--neo4j-http-tls-enabled"),
		HTTPTLSCertFile:                   mergeFlagValue(httpTLSCertFile, neo4jHTTPTLSCertFile, "--http-tls-cert-file", "--neo4j-http-tls-cert-file"),
		HTTPTLSKeyFile:                    mergeFlagValue(httpTLSKeyFile, neo4jHTTPTLSKeyFile, "--http-tls-key-file", "--neo4j-http-tls-key-file"),
		HTTPAllowUnauthenticatedPing:      mergeFlagValue(allowUnauthenticatedPing, neo4jHTTPAllowUnauthenticatedPing, "--http-allow-unauthenticated-ping", "--neo4j-http-allow-unauthenticated-ping"),
		HTTPAllowUnauthenticatedToolsList: mergeFlagValue(allowUnauthenticatedToolsList, neo4jHTTPAllowUnauthenticatedToolsList, "--http-allow-unauthenticated-tools-list", "--neo4j-http-allow-unauthenticated-tools-list"),
		AuthHeaderName:                    mergeFlagValue(authHeaderName, neo4jAuthHeaderName, "--http-auth-header-name", "--neo4j-http-auth-header-name"),
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
