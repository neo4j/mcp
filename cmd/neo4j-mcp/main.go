package main

import (
	"context"
	"fmt"
	"os"

	"github.com/neo4j/mcp/internal/cli"
	"github.com/neo4j/mcp/internal/config"
	"github.com/neo4j/mcp/internal/database"
	"github.com/neo4j/mcp/internal/logger"
	"github.com/neo4j/mcp/internal/server"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

var Version = "development"

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

	// Verify database connectivity
	if err := driver.VerifyConnectivity(ctx); err != nil {
		log.Error("Failed to verify database connectivity", "error", err)
		return
	}

	// Create database service
	dbService, err := database.NewNeo4jService(driver, cfg.Database)
	if err != nil {
		log.Error("Failed to create database service", "error", err)
		return
	}

	// Create and configure the MCP server
	mcpServer := server.NewNeo4jMCPServer(Version, cfg, dbService, log)

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
