package main

import (
	"github.com/neo4j/mcp/internal/server"
)

func main() {
	// Create and configure the MCP server
	mcpServer := server.NewNeo4jMCPServer()
	defer mcpServer.Stop()

	// Register all tools
	mcpServer.RegisterTools()

	mcpServer.Start()

}
