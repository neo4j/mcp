//go:build integration

package integration

import (
	"testing"

	"github.com/neo4j/mcp/internal/database"
	"github.com/neo4j/mcp/internal/tools/cypher"
	"github.com/neo4j/mcp/test/integration/container_runner"
	"github.com/neo4j/mcp/test/integration/helpers"
)

func TestWriteCypher(t *testing.T) {
	t.Parallel()
	driver := container_runner.GetContainerDriver()
	databaseService, err := database.NewNeo4jService(*driver, "neo4j")
	if err != nil {
		t.Fatalf("failed to create Neo4j service: %v", err)
	}
	tc := helpers.NewTestContext(t, databaseService)

	personLabel := tc.GetUniqueLabel("Person")

	write := cypher.WriteCypherHandler(tc.Deps)
	tc.CallTool(write, map[string]any{
		"query":  "CREATE (p:" + personLabel + " {name: $name}) RETURN p",
		"params": map[string]any{"name": "Alice"},
	})

	tc.VerifyNodeInDB(personLabel, map[string]any{"name": "Alice"})
}
