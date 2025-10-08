package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/neo4j/mcp/internal/config"
	"github.com/neo4j/mcp/internal/server"
)

var Version = "development"

func main() {
	// Handle version flag
	if len(os.Args) > 1 && os.Args[1] == "-v" {
		// NOTE: "standard" log package logger write on on STDERR, in this case we want explicitly to write to STDOUT
		fmt.Printf("neo4j-mcp version: %s\n", Version)
		return
	}

	// Handle help flag
	if len(os.Args) > 1 && (os.Args[1] == "-h" || os.Args[1] == "--help") {
		printHelp()
		return
	}

	// get config from environment variables
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Create and configure the MCP server
	mcpServer, err := server.NewNeo4jMCPServer(Version, cfg)
	if err != nil {
		log.Fatalf("Failed to create MCP server: %v", err)
	}

	// Gracefully handle shutdown
	ctx := context.Background()
	defer func() {
		if err := mcpServer.Stop(ctx); err != nil {
			log.Fatalf("Error stopping server: %v", err)
		}
	}()

	// Start the server (this blocks until the server is stopped)
	if err := mcpServer.Start(ctx); err != nil {
		log.Printf("Server error: %v", err)
		return // so that defer can run
	}
}

func printHelp() {
	fmt.Println("Neo4j MCP Server")
	fmt.Println("\nUsage:")
	fmt.Println("  neo4j-mcp [flags]")
	fmt.Println("\nFlags:")
	fmt.Println("  -v          Show version")
	fmt.Println("  -h, --help  Show this help message")
	fmt.Println("\nEnvironment Variables:")
	fmt.Println("  NEO4J_URI         Neo4j connection URI (default: bolt://localhost:7687)")
	fmt.Println("  NEO4J_USERNAME    Neo4j username (default: neo4j)")
	fmt.Println("  NEO4J_PASSWORD    Neo4j password (default: password)")
	fmt.Println("  NEO4J_DATABASE    Neo4j database name (default: neo4j)")
	fmt.Println("  MCP_TRANSPORT     Transport mode: 'stdio' or 'http' (default: stdio)")
	fmt.Println("\nHTTP Mode Environment Variables (when MCP_TRANSPORT=http):")
	fmt.Println("  MCP_HTTP_HOST     HTTP server host (default: localhost)")
	fmt.Println("  MCP_HTTP_PORT     HTTP server port (default: 8080)")
	fmt.Println("  MCP_HTTP_PATH     HTTP endpoint path (default: /mcp)")
	fmt.Println("\nExamples:")
	fmt.Println("  # Run in stdio mode (default)")
	fmt.Println("  neo4j-mcp")
	fmt.Println("\n  # Run in HTTP mode")
	fmt.Println("  MCP_TRANSPORT=http neo4j-mcp")
	fmt.Println("\n  # Run in HTTP mode on custom port")
	fmt.Println("  MCP_TRANSPORT=http MCP_HTTP_PORT=9000 neo4j-mcp")
}
