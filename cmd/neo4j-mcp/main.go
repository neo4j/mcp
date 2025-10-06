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
			log.Fatalf("Error closing driver: %v", err)
		}
	}()

	// Verify database connectivity
	if err := driver.VerifyConnectivity(ctx); err != nil {
		log.Fatalf("Failed to verify database connectivity: %v", err)
	}

	// Create database service
	dbService, err := database.NewNeo4jService(driver)
	if err != nil {
		log.Fatalf("Failed to create database service: %v", err)
	}

	// Create and configure the MCP server
	mcpServer := server.NewNeo4jMCPServer(Version, cfg, dbService)

	// Gracefully handle shutdown
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
