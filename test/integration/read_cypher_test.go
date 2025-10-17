package integration_test

import (
	"testing"

	"github.com/neo4j/mcp/internal/tools"
)

func TestMCPIntegration_ReadCypher(t *testing.T) {
	t.Parallel()

	tc := NewTestContext(t)

	if err := tc.SeedNode("Person", map[string]any{"name": "Alice"}); err != nil {
		t.Fatalf("failed to seed data: %v", err)
	}

	read := tools.ReadCypherHandler(tc.Deps)
	res := tc.CallTool(read, map[string]any{
		"query":  "MATCH (p:Person {name: $name, test_id: $testID}) RETURN p",
		"params": map[string]any{"name": "Alice", "testID": tc.TestID},
	})

	var records []map[string]any
	tc.ParseJSONResponse(res, &records)

	if len(records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(records))
	}

	pNode := records[0]["p"].(map[string]any)
	AssertNodeProperties(t, pNode, map[string]any{"name": "Alice"})
	AssertNodeHasLabel(t, pNode, "Person")
}
