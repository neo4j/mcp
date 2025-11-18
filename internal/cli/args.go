package cli

import (
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
  --neo4j-uri <URI>                   Neo4j connection URI (overrides env var)
  --neo4j-username <USERNAME>         Database username (overrides env var)
  --neo4j-password <PASSWORD>         Database password (overrides env var)
  --neo4j-database <DATABASE>         Database name (overrides env var)

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

/*
Example walkthrough for argument parsing:

neo4j-mcp --neo4j-uri bolt://localhost:7687 --neo4j-username test

os.Args:
- os.Args[0] = "neo4j-mcp"
- os.Args[1] = "--neo4j-uri"
- os.Args[2] = "bolt://localhost:7687"
- os.Args[3] = "--neo4j-username"
- os.Args[4] = "test"

As the loop processes:
1. i=1: Matches case "--neo4j-uri" → i += 2 → i=3 (skips the URI value)
2. i=3: Matches case "--neo4j-username" → i += 2 → i=5 (skips the "test" value)
3. i=5: Loop ends

This allows the arguments to "pass through" untouched so that flag.Parse() in main.go can later handle them properly.
*/

// HandleArgs processes command-line arguments for version and help flags.
// It exits the program after displaying the requested information.
// If unknown flags are encountered, it prints an error message and exits.
// Known configuration flags are skipped to allow the flag package to handle them.
// Why are we skipping known configuration flags?
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
		case "--neo4j-uri", "--neo4j-username", "--neo4j-password", "--neo4j-database":
			// Check if there's a value following the flag
			if i+1 >= len(os.Args) {
				err = fmt.Errorf("%s requires a value", arg)
				break
			}
			// Check if next argument is another flag (starts with --)
			nextArg := os.Args[i+1]
			if strings.HasPrefix(nextArg, "--") {
				err = fmt.Errorf("%s requires a value (got flag %s instead)", arg, nextArg)
				break
			}
			// Safe to skip flag and value - let flag package handle them
			i += 2
		default:
			if arg == "--" {
				// Stop processing our flags, let flag package handle the rest
				i = len(os.Args) // Exit the loop
				break
			}
			err = fmt.Errorf("unknown flag or argument: %s", arg)
			i++
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
