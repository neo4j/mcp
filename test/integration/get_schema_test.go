//go:build integration

package integration

import (
	"testing"

	"github.com/neo4j/mcp/internal/tools"
)

func TestMCPIntegration_GetSchema(t *testing.T) {
	t.Parallel()

	tc := NewTestContext(t)

	if err := tc.SeedNode("Person", map[string]any{"name": "Alice", "age": 30}); err != nil {
		t.Fatalf("failed to seed Person node: %v", err)
	}
	if err := tc.SeedNode("Company", map[string]any{"name": "Neo4j", "founded": 2007}); err != nil {
		t.Fatalf("failed to seed Company node: %v", err)
	}

	getSchema := tools.GetSchemaHandler(tc.Deps)
	res := tc.CallTool(getSchema, nil)

	var schemaArray []map[string]any
	tc.ParseJSONResponse(res, &schemaArray)

	if len(schemaArray) == 0 {
		t.Fatal("expected schema to contain at least one entry")
	}

	schemaMap := make(map[string]map[string]any)
	for _, entry := range schemaArray {
		key, ok := entry["key"].(string)
		if !ok {
			continue
		}
		value, ok := entry["value"].(map[string]any)
		if !ok {
			continue
		}
		schemaMap[key] = value
	}

	AssertSchemaHasNodeType(t, schemaMap, "Person", []string{"name", "age"})
	AssertSchemaHasNodeType(t, schemaMap, "Company", []string{"name", "founded"})
}
