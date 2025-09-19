package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/neo4j/mcp/internal/config"
	"github.com/neo4j/mcp/internal/server"
)

func main() {
	// Handle version flag
	if len(os.Args) > 1 && os.Args[1] == "-v" {
		// NOTE: "standard" log package logger write on on STDERR, in this case we want explicitly to write to STDOUT
		fmt.Printf("neo4j-mcp version: %s\n", server.Version)
		return
	}
	// get config from environment variables
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Create and configure the MCP server
	mcpServer, err := server.NewNeo4jMCPServer(cfg)
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
		log.Fatalf("Server error: %v", err)
	}
}
