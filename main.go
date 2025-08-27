package main

import (
	"github.com/neo4j/mcp/internal/config"
	"github.com/neo4j/mcp/internal/server"
)

func main() {
	// get config from environment variables
	cfg := config.LoadConfig()

	// Create and configure the MCP server
	mcpServer := server.NewNeo4jMCPServer(cfg)
	defer mcpServer.Stop()

	// Register all tools
	mcpServer.RegisterTools()

	mcpServer.Start()

}
