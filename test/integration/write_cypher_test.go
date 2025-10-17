//go:build integration

package integration_test

import (
	"testing"

	"github.com/neo4j/mcp/internal/tools"
)

func TestMCPIntegration_WriteCypher(t *testing.T) {
	t.Parallel()

	tc := NewTestContext(t)

	write := tools.WriteCypherHandler(tc.Deps)
	tc.CallTool(write, map[string]any{
		"query":  "CREATE (p:Person {name: $name, test_id: $testID}) RETURN p",
		"params": map[string]any{"name": "Alice", "testID": tc.TestID},
	})

	tc.VerifyNodeInDB("Person", map[string]any{"name": "Alice"})
}
