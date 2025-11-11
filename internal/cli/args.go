package cli

import (
	"fmt"
	"os"
)

// osExit is a variable that can be mocked in tests
var osExit = os.Exit

const helpText = `neo4j-mcp - Neo4j Model Context Protocol Server

Usage:
  neo4j-mcp [OPTIONS]

Options:
  -h, --help      Show this help message
  -v, --version   Show version information

Environment Variables:
  NEO4J_URI       Neo4j database URI (default: bolt://localhost:7687)
  NEO4J_USERNAME  Database username (default: neo4j)
  NEO4J_PASSWORD  Database password (required)
  NEO4J_DATABASE  Database name (default: neo4j)

Examples:
  NEO4J_PASSWORD=mypassword neo4j-mcp
  NEO4J_URI=bolt://db.example.com:7687 NEO4J_USERNAME=admin NEO4J_PASSWORD=secret neo4j-mcp

For more information, visit: https://github.com/neo4j/mcp
`

func HandleArgs(version string) {
	if len(os.Args) <= 1 {
		return
	}

	flags := make(map[string]bool)
	var err error

	for _, arg := range os.Args[1:] {
		switch arg {
		case "-h", "--help":
			flags["help"] = true
		case "-v", "--version":
			flags["version"] = true
		default:
			err = fmt.Errorf("unknown flag or argument: %s", arg)
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
