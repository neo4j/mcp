package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/neo4j/mcp/internal/config"
	"github.com/neo4j/mcp/internal/database"
	"github.com/neo4j/mcp/internal/server"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
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

	// Initialize Neo4j driver
	driver, err := neo4j.NewDriverWithContext(cfg.URI, neo4j.BasicAuth(cfg.Username, cfg.Password, ""))
	if err != nil {
		log.Fatalf("Failed to create Neo4j driver: %v", err)
	}

	// Gracefully handle shutdown
	ctx := context.Background()
	defer func() {
		if err := driver.Close(ctx); err != nil {
			log.Printf("Error closing driver: %v", err)
		}
	}()

	// Verify database connectivity
	if err := driver.VerifyConnectivity(ctx); err != nil {
		log.Printf("Failed to verify database connectivity: %v", err)
		return
	}

	// Create database service
	dbService, err := database.NewNeo4jService(driver)
	if err != nil {
		log.Printf("Failed to create database service: %v", err)
		return
	}

	// Create and configure the MCP server
	mcpServer := server.NewNeo4jMCPServer(Version, cfg, dbService)

	// Gracefully handle shutdown
	defer func() {
		if err := mcpServer.Stop(); err != nil {
			log.Printf("Error stopping server: %v", err)
		}
	}()

	// Start the server (this blocks until the server is stopped)
	if err := mcpServer.Start(); err != nil {
		log.Printf("Server error: %v", err)
		return // so that defer can run
	}
}

func printHelp() {
	log.Printf("Neo4j MCP Server")
	log.Printf("\nUsage:")
	log.Printf("  neo4j-mcp [flags]")
	log.Printf("\nFlags:")
	log.Printf("  -v          Show version")
	log.Printf("  -h, --help  Show this help message")
	log.Printf("\nEnvironment Variables:")
	log.Printf("  NEO4J_URI         Neo4j connection URI (default: bolt://localhost:7687)")
	log.Printf("  NEO4J_USERNAME    Neo4j username (default: neo4j)")
	log.Printf("  NEO4J_PASSWORD    Neo4j password (default: password)")
	log.Printf("  NEO4J_DATABASE    Neo4j database name (default: neo4j)")
	log.Printf("  MCP_TRANSPORT     Transport mode: 'stdio' or 'http' (default: stdio)")
	log.Printf("\nHTTP Mode Environment Variables (when MCP_TRANSPORT=http):")
	log.Printf("  MCP_HTTP_HOST     HTTP server host (default: 127.0.0.1)")
	log.Printf("  MCP_HTTP_PORT     HTTP server port (default: 8080)")
	log.Printf("  MCP_HTTP_PATH     HTTP endpoint path (default: /mcp)")
	log.Printf("\nExamples:")
	log.Printf("  # Run in stdio mode (default)")
	log.Printf("  neo4j-mcp")
	log.Printf("\n  # Run in HTTP mode")
	log.Printf("  MCP_TRANSPORT=http neo4j-mcp")
	log.Printf("\n  # Run in HTTP mode on custom port")
	log.Printf("  MCP_TRANSPORT=http MCP_HTTP_PORT=9000 neo4j-mcp")
}
