package main

import (
	"context"
	"flag"
	"log"
	"strings"

	"github.com/neo4j/mcp/internal/analytics"
	"github.com/neo4j/mcp/internal/cli"
	"github.com/neo4j/mcp/internal/config"
	"github.com/neo4j/mcp/internal/database"
	"github.com/neo4j/mcp/internal/server"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

// go build -C cmd/neo4j-mcp -o ../../bin/ -ldflags "-X 'main.Version=9999' -X 'main.MixPanelEndpoint=https://api-eu.mixpanel.com' -X 'main.MixPanelToken=your-mixpanel-token'"
var Version = "development"
var MixPanelEndpoint = ""
var MixPanelToken = ""

func main() {
	// Define CLI flags for configuration
	neo4jURI := flag.String("neo4j-uri", "", "Neo4j connection URI (overrides NEO4J_URI env var)")
	neo4jUsername := flag.String("neo4j-username", "", "Neo4j username (overrides NEO4J_USERNAME env var)")
	neo4jPassword := flag.String("neo4j-password", "", "Neo4j password (overrides NEO4J_PASSWORD env var)")
	neo4jDatabase := flag.String("neo4j-database", "", "Neo4j database name (overrides NEO4J_DATABASE env var)")

	// Handle CLI arguments (version, help, etc.)
	cli.HandleArgs(Version)

	// Parse flags after HandleArgs
	flag.Parse()

	// Load config from environment variables
	cfg := config.LoadConfig()

	// Override config with CLI flags if provided (CLI flags take precedence)
	if *neo4jURI != "" {
		cfg.URI = *neo4jURI
	}
	if *neo4jUsername != "" {
		cfg.Username = *neo4jUsername
	}
	if *neo4jPassword != "" {
		cfg.Password = *neo4jPassword
	}
	if *neo4jDatabase != "" {
		cfg.Database = *neo4jDatabase
	}

	// Validate configuration after applying CLI overrides
	if err := cfg.Validate(); err != nil {
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
	isAura := strings.Contains(cfg.URI, "database.neo4j.io")
	anService := analytics.NewAnalytics(MixPanelToken, MixPanelEndpoint, isAura)

	if cfg.Telemetry == "false" || MixPanelEndpoint == "" || MixPanelToken == "" {
		log.Println("Telemetry disabled.")
		anService.Disable()
	} else if cfg.Telemetry == "true" {
		anService.Enable()
		log.Println("Telemetry is enabled to help us improve the product by collecting anonymous usage data such as: tools being used, the operating system, and CPU architecture.")
		log.Println("To disable telemetry, set the NEO4J_TELEMETRY environment variable to \"false\".")
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
