package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/neo4j/mcp/internal/analytics"
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
			log.Printf("Error closing driver: %v", err)
		}
	}()

	// Verify database connectivity
	if err := driver.VerifyConnectivity(ctx); err != nil {
		log.Printf("Failed to verify database connectivity: %v", err)
		return
	}

	// Create database service
	dbService, err := database.NewNeo4jService(driver, cfg.Database)
	if err != nil {
		log.Printf("Failed to create database service: %v", err)
		return
	}
	anService := analytics.NewAnalytics(cfg.MixPanelToken, cfg.MixPanelEndpoint)

	// initialize the analytics
	if cfg.Telemetry == "true" {
		anService.Enable()
		log.Println("Telemetry is enabled to help us improve the product by collecting anonymous usage data such as: tools in use, OS and CPU architecture.")
		log.Println("To disable telemetry, set the NEO4J_TELEMETRY environment variable to \"false\".")
	} else {
		anService.Disable()
	}

	// Create and configure the MCP server
	mcpServer := server.NewNeo4jMCPServer(Version, cfg, dbService, anService)

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
