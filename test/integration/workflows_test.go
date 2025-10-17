package integration_test

import (
	"testing"

	"github.com/neo4j/mcp/internal/tools"
)

func TestMCPIntegration_WriteThenRead(t *testing.T) {
	t.Parallel()

	tc := NewTestContext(t)

	write := tools.WriteCypherHandler(tc.Deps)
	tc.CallTool(write, map[string]any{
		"query":  "CREATE (c:Company {name: $name, industry: $industry, test_id: $testID}) RETURN c",
		"params": map[string]any{"name": "Neo4j", "industry": "Database", "testID": tc.TestID},
	})

	read := tools.ReadCypherHandler(tc.Deps)
	res := tc.CallTool(read, map[string]any{
		"query":  "MATCH (c:Company {test_id: $testID}) RETURN c",
		"params": map[string]any{"testID": tc.TestID},
	})

	var records []map[string]any
	tc.ParseJSONResponse(res, &records)

	if len(records) != 1 {
		t.Fatalf("expected 1 company, got %d", len(records))
	}

	company := records[0]["c"].(map[string]any)
	AssertNodeProperties(t, company, map[string]any{
		"name":     "Neo4j",
		"industry": "Database",
	})
	AssertNodeHasLabel(t, company, "Company")
}
