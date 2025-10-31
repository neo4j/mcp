//go:build integration

package integration

import (
	"testing"

	"github.com/neo4j/mcp/internal/tools/cypher"
	"github.com/neo4j/mcp/test/integration/helpers"
)

func TestGetSchema(t *testing.T) {
	// do not run this test in Parallel, test-id will not avoid collision when testing the emptiness of the db.
	t.Run("get-schema should collect information about the neo4j database", func(t *testing.T) {
		tc := helpers.NewTestContext(t, dbs.GetDriver())

		// Use TestID as identifier to create unique labels
		personLabel, err := tc.SeedNode("Person", map[string]any{"name": "Alice", "age": 30})
		if err != nil {
			t.Fatalf("failed to seed Person node: %v", err)
		}
		companyLabel, err := tc.SeedNode("Company", map[string]any{"name": "Neo4j", "founded": 2007})
		if err != nil {
			t.Fatalf("failed to seed Company node: %v", err)
		}

		getSchema := cypher.GetSchemaHandler(tc.Deps)
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

		// Check for the unique labels created
		tc.AssertSchemaHasNodeType(schemaMap, personLabel, []string{"name", "age"})
		tc.AssertSchemaHasNodeType(schemaMap, companyLabel, []string{"name", "founded"})
	})

	t.Run("get-schema should give hint when the database is empty", func(t *testing.T) {
		tc := helpers.NewTestContext(t, dbs.GetDriver())

		getSchema := cypher.GetSchemaHandler(tc.Deps)
		res := tc.CallTool(getSchema, nil)

		textContent := tc.ParseTextResponse(res)

		expectedMessage := "The get-schema tool executed successfully; however, since the Neo4j instance contains no data, no schema information was returned."
		if textContent != expectedMessage {
			t.Fatal("no empty schema hint returned")
		}
	})
}
