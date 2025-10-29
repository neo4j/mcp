//go:build integration

package integration

import (
	"testing"

	"github.com/neo4j/mcp/internal/tools/cypher"
	"github.com/neo4j/mcp/test/integration/container_runner"
	"github.com/neo4j/mcp/test/integration/helpers"
)

func TestWriteThenRead(t *testing.T) {
	t.Parallel()
	tc := helpers.NewTestContext(t, container_runner.GetContainerDriver())

	companyLabel := tc.GetUniqueLabel("Company")

	write := cypher.WriteCypherHandler(tc.Deps)
	tc.CallTool(write, map[string]any{
		"query":  "CREATE (c:" + companyLabel + " {name: $name, industry: $industry}) RETURN c",
		"params": map[string]any{"name": "Neo4j", "industry": "Database"},
	})

	read := cypher.ReadCypherHandler(tc.Deps)
	res := tc.CallTool(read, map[string]any{
		"query":  "MATCH (c:" + companyLabel + ") RETURN c",
		"params": map[string]any{},
	})

	var records []map[string]any
	tc.ParseJSONResponse(res, &records)

	if len(records) != 1 {
		t.Fatalf("expected 1 company, got %d", len(records))
	}

	company := records[0]["c"].(map[string]any)
	helpers.AssertNodeProperties(t, company, map[string]any{
		"name":     "Neo4j",
		"industry": "Database",
	})
	helpers.AssertNodeHasLabel(t, company, companyLabel)
}
