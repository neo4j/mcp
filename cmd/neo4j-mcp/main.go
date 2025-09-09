package main

import (
	"context"
	"log"

	"github.com/neo4j/mcp/internal/config"
	"github.com/neo4j/mcp/internal/server"
)

func main() {
	// get config from environment variables
	cfg := config.LoadConfig()

	// Create and configure the MCP server
	mcpServer, err := server.NewNeo4jMCPServer(cfg)
	if err != nil {
		log.Fatalf("Failed to create MCP server: %v", err)
	}

	// Gracefully handle shutdown
	ctx := context.Background()
	defer func() {
		if err := mcpServer.Stop(ctx); err != nil {
			log.Printf("Error stopping server: %v", err)
		}
	}()

	// Start the server (this blocks until the server is stopped)
	if err := mcpServer.Start(ctx); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
