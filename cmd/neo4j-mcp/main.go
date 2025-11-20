package main

import (
	"context"
	"log"

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
	// Handle CLI arguments (version, help, etc.)
	cli.HandleArgs(Version)

	// Parse CLI flags for configuration
	cliArgs := cli.ParseConfigFlags()

	// Load and validate configuration (env vars + CLI overrides)
	cfg, err := config.LoadConfig(&config.CLIOverrides{
		URI:       cliArgs.URI,
		Username:  cliArgs.Username,
		Password:  cliArgs.Password,
		Database:  cliArgs.Database,
		ReadOnly:  cliArgs.ReadOnly,
		Telemetry: cliArgs.Telemetry,
	})
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

	// Create database service
	dbService, err := database.NewNeo4jService(driver, cfg.Database)
	if err != nil {
		log.Printf("Failed to create database service: %v", err)
		return
	}

	anService := analytics.NewAnalytics(MixPanelToken, MixPanelEndpoint, cfg.URI)

	// Enable telemetry only when user has opted in AND the required tokens are present
	if cfg.Telemetry && MixPanelEndpoint != "" && MixPanelToken != "" {
		anService.Enable()
		log.Println("Telemetry is enabled to help us improve the product by collecting anonymous usage data such as: tools being used, the operating system, and CPU architecture.")
		log.Println("To disable telemetry, set the NEO4J_TELEMETRY environment variable to \"false\".")
	} else {
		log.Println("Telemetry disabled.")
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
