package main

import (
	"context"
	"fmt"
	"os"

	"github.com/neo4j/mcp/internal/analytics"
	"github.com/neo4j/mcp/internal/cli"
	"github.com/neo4j/mcp/internal/config"
	"github.com/neo4j/mcp/internal/database"
	"github.com/neo4j/mcp/internal/logger"
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

	// get config from environment variables
	cfg, err := config.LoadConfig()
	if err != nil {
		// Can't use logger here yet, so just print to stderr
		fmt.Println("Failed to load configuration: " + err.Error())
		os.Exit(1)
	}

	// Initialize logger
	log := logger.New(cfg.LogLevel, cfg.LogFormat, os.Stderr)

	// Initialize Neo4j driver
	driver, err := neo4j.NewDriverWithContext(cfg.URI, neo4j.BasicAuth(cfg.Username, cfg.Password, ""))
	if err != nil {
		log.Error("Failed to create Neo4j driver", "error", err)
		os.Exit(1)
	}

	// Gracefully handle shutdown
	ctx := context.Background()
	defer func() {
		if err := driver.Close(ctx); err != nil {
			log.Error("Error closing driver", "error", err)
		}
	}()

	// Create database service
	dbService, err := database.NewNeo4jService(driver, cfg.Database, log)
	if err != nil {
		log.Error("Failed to create database service", "error", err)
		return
	}

	anService := analytics.NewAnalytics(MixPanelToken, MixPanelEndpoint, cfg.URI)

	if cfg.Telemetry == "false" || MixPanelEndpoint == "" || MixPanelToken == "" {
		log.Info("Telemetry disabled.")
		anService.Disable()
	} else if cfg.Telemetry == "true" {
		anService.Enable()
		log.Info("Telemetry is enabled to help us improve the product by collecting anonymous usage data " +
			"such as: tools being used, the operating system, and CPU architecture.\n" +
			"To disable telemetry, set the NEO4J_TELEMETRY environment variable to \"false\".")
	}

	// Create and configure the MCP server
	mcpServer := server.NewNeo4jMCPServer(Version, cfg, dbService, anService, log)

	// Gracefully handle shutdown
	defer func() {
		if err := mcpServer.Stop(); err != nil {
			log.Error("Error stopping server", "error", err)
		}
	}()

	// Start the server (this blocks until the server is stopped)
	if err := mcpServer.Start(); err != nil {
		log.Error("Server error", "error", err)
		return // so that defer can run
	}

}
