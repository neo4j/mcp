//go:build integration

package integration

import (
	"testing"

	"github.com/neo4j/mcp/internal/tools/cypher"
	"github.com/neo4j/mcp/test/integration/helpers"
)

// TODO: Add more tests. The current tests showcase how to use the integration test framework, but are not exhaustive.

func TestMCPIntegration_WriteCypher(t *testing.T) {
	t.Parallel()

	tc := helpers.NewTestContext(t)

	write := cypher.WriteCypherHandler(tc.Deps)
	tc.CallTool(write, map[string]any{
		"query":  "CREATE (p:Person {name: $name, test_id: $testID}) RETURN p",
		"params": map[string]any{"name": "Alice", "testID": tc.TestID},
	})

	tc.VerifyNodeInDB("Person", map[string]any{"name": "Alice"})
}
