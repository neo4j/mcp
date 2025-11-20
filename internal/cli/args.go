package cli

import (
	"flag"
	"fmt"
	"os"
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

Required Environment Variables:
  NEO4J_URI       Neo4j database URI
  NEO4J_USERNAME  Database username
  NEO4J_PASSWORD  Database password

Optional Environment Variables:
  NEO4J_DATABASE  Database name (default: neo4j)
  NEO4J_TELEMETRY Enable/disable telemetry (default: true)
  NEO4J_READ_ONLY Enable read-only mode (default: false)

Examples:
  # Using environment variables
  NEO4J_URI=bolt://localhost:7687 NEO4J_USERNAME=neo4j NEO4J_PASSWORD=password neo4j-mcp

  # Using CLI flags (takes precedence over environment variables)
  neo4j-mcp --neo4j-uri bolt://localhost:7687 --neo4j-username neo4j --neo4j-password password

For more information, visit: https://github.com/neo4j/mcp
`

// Args holds configuration values parsed from command-line flags
type Args struct {
	URI       string
	Username  string
	Password  string
	Database  string
	ReadOnly  string
	Telemetry string
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

	flag.Parse()

	return &Args{
		URI:       *neo4jURI,
		Username:  *neo4jUsername,
		Password:  *neo4jPassword,
		Database:  *neo4jDatabase,
		ReadOnly:  *neo4jReadOnly,
		Telemetry: *neo4jTelemetry,
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
		switch arg {
		case "-h", "--help":
			flags["help"] = true
			i++
		case "-v", "--version":
			flags["version"] = true
			i++
		// Allow configuration flags to be parsed by the flag package
		case "--neo4j-uri", "--neo4j-username", "--neo4j-password", "--neo4j-database", "--neo4j-read-only", "--neo4j-telemetry":
			// Check if there's a value following the flag
			if i+1 >= len(os.Args) {
				err = fmt.Errorf("%s requires a value", arg)
				break
			}
			// Check if next argument is another flag (starts with --)
			nextArg := os.Args[i+1]
			if strings.HasPrefix(nextArg, "-") {
				err = fmt.Errorf("%s requires a value (got flag %s instead)", arg, nextArg)
				break
			}
			// Safe to skip flag and value - let flag package handle them
			i += 2
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
