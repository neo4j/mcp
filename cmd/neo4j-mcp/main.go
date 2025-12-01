package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

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

	// Parse CLI flags for configuration
	cliArgs := cli.ParseConfigFlags()

	// Load and validate configuration (env vars + CLI overrides)
	cfg, err := config.LoadConfig(&config.CLIOverrides{
		URI:           cliArgs.URI,
		Username:      cliArgs.Username,
		Password:      cliArgs.Password,
		Database:      cliArgs.Database,
		ReadOnly:      cliArgs.ReadOnly,
		Telemetry:     cliArgs.Telemetry,
		TransportMode: cliArgs.TransportMode,
	})
	if err != nil {
		// Can't use logger here yet, so just print to stderr
		fmt.Fprintln(os.Stderr, "Failed to load configuration: "+err.Error())
		os.Exit(1)
	}

	// Initialize global logger
	logger.Init(cfg.LogLevel, cfg.LogFormat, os.Stderr)

	// Initialize Neo4j driver
	driver, err := neo4j.NewDriverWithContext(cfg.URI, neo4j.BasicAuth(cfg.Username, cfg.Password, ""))
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
	dbService, err := database.NewNeo4jService(driver, cfg.Database, cfg.URI)
	if err != nil {
		slog.Error("Failed to create database service", "error", err)
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

	if cfg.TransportMode == config.TransportModeHTTP {
		// Run Start in a goroutine so we can handle signals concurrently.
		errChan := make(chan error, 1)
		go func() {
			errChan <- mcpServer.Start()
		}()

		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

		select {
		case sig := <-sigChan:
			slog.Info("Shutdown signal received", "signal", sig.String())
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			if err := mcpServer.Stop(shutdownCtx); err != nil {
				slog.Error("Error stopping server", "error", err)
			}
		case err := <-errChan:
			if err != nil {
				slog.Error("Server error", "error", err)
			}
		}
	} else {
		// stdio mode blocks; when it returns we simply exit.
		if err := mcpServer.Start(); err != nil {
			slog.Error("Server error", "error", err)
		}
	}
}
