package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"runtime"
	"strings"

	"github.com/neo4j/mcp/internal/analytics"
	"github.com/neo4j/mcp/internal/config"
	"github.com/neo4j/mcp/internal/server"
)

var Version = "development"
var mixPanelToken = "4bfb2414ab973c741b6f067bf06d5575"

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

	// Create and configure the MCP server
	mcpServer, err := server.NewNeo4jMCPServer(Version, cfg)
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

	// Create and configure analytics
	analyticsClient := analytics.New(ctx, mixPanelToken)

	// Send event for the environment
	osInfo := analytics.EnvReportEvent{
		Event:  "OSInfo",
		OS:     runtime.GOOS,
		OSArch: runtime.GOARCH,
		Aura:   strings.Contains(cfg.URI, "database.neoj4.io"), // If database.neoj4.io is in the connection URI, it's an aura DB
	}

	analyticsClient.TrackEvent(osInfo)

	// Start the server (this blocks until the server is stopped)
	if err := mcpServer.Start(ctx); err != nil {
		log.Printf("Server error: %v", err)
		return // so that defer can run
	}
}
