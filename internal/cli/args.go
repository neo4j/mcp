package cli

import (
	"fmt"
	"os"
)

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

// HandleArgs processes command-line arguments and returns true if the program should exit early
// (e.g., after showing help or version), or false if it should continue to normal execution.
func HandleArgs(version string) bool {
	if len(os.Args) <= 1 {
		return false
	}

	arg := os.Args[1]

	switch arg {
	case "-v", "--version":
		// NOTE: "standard" log package logger write on on STDERR, in this case we want explicitly to write to STDOUT
		fmt.Printf("neo4j-mcp version: %s\n", version)
		return true
	case "-h", "--help":
		fmt.Print(helpText)
		return true
	}

	return false
}
