package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"

	"github.com/neo4j/mcp/internal/analytics"
	"github.com/neo4j/mcp/internal/cli"
	"github.com/neo4j/mcp/internal/config"
	"github.com/neo4j/mcp/internal/database"
	"github.com/neo4j/mcp/internal/logger"
	"github.com/neo4j/mcp/internal/server"
	"github.com/neo4j/neo4j-go-driver/v6/neo4j"
)

// go build -C cmd/neo4j-mcp -o ../../bin/ -ldflags "-X 'main.Version=9999'"
var Version = "development"

const MixPanelEndpoint = "https://api.mixpanel.com"
const MixPanelToken = "4bfb2414ab973c741b6f067bf06d5575" // #nosec G101 -- MixPanel tokens are safe to be public

func main() {
	// Handle CLI arguments (version, help, etc.)
	cli.HandleArgs(Version)

	// Parse CLI flags for configuration
	cliArgs := cli.ParseConfigFlags()

	// Load and validate configuration (env vars + CLI overrides)
	cfg, err := config.LoadConfig(&config.CLIOverrides{
		URI:                      cliArgs.URI,
		Username:                 cliArgs.Username,
		Password:                 cliArgs.Password,
		Database:                 cliArgs.Database,
		ReadOnly:                 cliArgs.ReadOnly,
		Telemetry:                cliArgs.Telemetry,
		TransportMode:            cliArgs.TransportMode,
		Port:                     cliArgs.HTTPPort,
		Host:                     cliArgs.HTTPHost,
		AllowedOrigins:           cliArgs.HTTPAllowedOrigins,
		TLSEnabled:               cliArgs.HTTPTLSEnabled,
		TLSCertFile:              cliArgs.HTTPTLSCertFile,
		TLSKeyFile:               cliArgs.HTTPTLSKeyFile,
		AuthHeaderName:           cliArgs.AuthHeaderName,
		AllowUnauthenticatedPing: cliArgs.HTTPAllowUnauthenticatedPing,
	})
	if err != nil {
		// Can't use logger here yet, so just print to stderr
		fmt.Fprintln(os.Stderr, "Failed to load configuration: "+err.Error())
		os.Exit(1)
	}

	// Initialize global logger
	logger.Init(cfg.LogLevel, cfg.LogFormat, os.Stderr)

	// Initialize Neo4j driver
	// For STDIO mode: use environment credentials
	// For HTTP mode: create driver without auth, per-request credentials will be used via impersonation
	// Credentials come from per-request Basic Auth headers
	var authToken neo4j.AuthToken
	if cfg.TransportMode == config.TransportModeStdio {
		authToken = neo4j.BasicAuth(cfg.Username, cfg.Password, "")
	}

	driver, err := neo4j.NewDriver(cfg.URI, authToken)
	if err != nil {
		slog.Error("Failed to create Neo4j driver", "error", err)
		os.Exit(1)
	}

	// Gracefully handle shutdown
	ctx := context.Background()
	defer func() {
		if err := driver.Close(ctx); err != nil {
			slog.Error("Error closing driver", "error", err)
		}
	}()

	// Create database service
	dbService, err := database.NewNeo4jService(driver, cfg.Database, cfg.TransportMode, Version)
	if err != nil {
		slog.Error("Failed to create database service", "error", err)
		return
	}

	anService := analytics.NewAnalytics(MixPanelToken, MixPanelEndpoint, cfg.URI)

	// Enable telemetry only when user has opted in AND Version is different from "development", which is changed via ldflags at build time.
	if cfg.Telemetry && Version != "development" {
		anService.Enable()
		log.Println("Telemetry is enabled to help us improve the product by collecting anonymous usage data such as: tools being used, the operating system, and CPU architecture.")
		log.Println("To disable telemetry, set the NEO4J_TELEMETRY environment variable to \"false\".")
	} else {
		log.Println("Telemetry disabled.")
		anService.Disable()
	}

	// Create and configure the MCP server
	mcpServer := server.NewNeo4jMCPServer(Version, cfg, dbService, anService)

	// Start the server - this blocks until shutdown for both stdio and HTTP modes
	if err := mcpServer.Start(); err != nil {
		slog.Error("Server error", "error", err)
		return
	}
}
