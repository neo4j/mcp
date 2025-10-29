//go:build integration

package integration

import (
	"testing"

	"github.com/neo4j/mcp/internal/database"
	"github.com/neo4j/mcp/internal/tools/cypher"
	"github.com/neo4j/mcp/test/integration/container_runner"
	"github.com/neo4j/mcp/test/integration/helpers"
)

func TestReadCypher(t *testing.T) {
	t.Parallel()
	driver := container_runner.GetContainerDriver()
	databaseService, err := database.NewNeo4jService(*driver, "neo4j")
	if err != nil {
		t.Fatalf("failed to create Neo4j service: %v", err)
	}
	tc := helpers.NewTestContext(t, databaseService)

	personLabel, err := tc.SeedNode("Person", map[string]any{"name": "Alice"})
	if err != nil {
		t.Fatalf("failed to seed data: %v", err)
	}

	read := cypher.ReadCypherHandler(tc.Deps)
	res := tc.CallTool(read, map[string]any{
		"query":  "MATCH (p:" + personLabel + " {name: $name}) RETURN p",
		"params": map[string]any{"name": "Alice"},
	})

	var records []map[string]any
	tc.ParseJSONResponse(res, &records)

	if len(records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(records))
	}

	pNode, ok := records[0]["p"].(map[string]any)
	if !ok {
		t.Fatalf("expected p to be map[string]any, got %T",
			records[0]["p"])
	}
	helpers.AssertNodeProperties(t, pNode, map[string]any{"name": "Alice"})
	helpers.AssertNodeHasLabel(t, pNode, personLabel)
}
