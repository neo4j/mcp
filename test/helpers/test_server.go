package helpers

import (
	"testing"

	"github.com/neo4j/mcp/internal/config"
	"github.com/neo4j/mcp/internal/database"
	"github.com/neo4j/mcp/internal/server"
)

// create an MCP server with test services injected.
func GetTestNeo4jMCPServer(cfg *config.Config, dbService *database.Neo4jService, t *testing.T) *server.Neo4jMCPServer {
	version := "test_server"

	// Create and configure the MCP server
	return server.NewNeo4jMCPServer(version, cfg, dbService, GetAnalyticsMock(t))
}
