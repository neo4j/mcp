//go:build integration

package integration

import (
	"slices"
	"testing"

	"github.com/neo4j/mcp/internal/tools/cypher"
	"github.com/neo4j/mcp/test/integration/helpers"
)

type SchemaPropertyType struct {
	Array     bool   `json:"array"`
	Existence bool   `json:"existence"`
	Indexed   bool   `json:"indexed"`
	Type      string `json:"type"`
}

type RelationshipType struct {
	Count      int                           `json:"count"`
	Direction  string                        `json:"direction"`
	Labels     []string                      `json:"labels"`
	Properties map[string]SchemaPropertyType `json:"properties"`
}

// SchemaValue represents the guaranteed structure inside the "value" field.
type SchemaValue struct {
	Properties    map[string]SchemaPropertyType `json:"properties"`
	Relationships map[string]RelationshipType   `json:"relationships"`
	Type          string                        `json:"type"`
}

// SchemaEntry represents one element of the returned by get-schema :
// { "key": "...", "value": { ... } }
type SchemaEntry struct {
	Key   string      `json:"key"`
	Value SchemaValue `json:"value"`
}

func TestGetSchema(t *testing.T) {
	// do not run this test in Parallel, test-id will not avoid collision when testing the emptiness of the db.
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

		var schemaEntries []SchemaEntry
		tc.ParseJSONResponse(res, &schemaEntries)

		if len(schemaEntries) == 0 {
			t.Fatal("expected schema to contain at least one entry")
		}
		assertSchemaHasLabel(t, schemaEntries, personLabel.String())
		assertSchemaHasLabel(t, schemaEntries, companyLabel.String())

		personEntry := getSchemaEntryByTypeOrLabel(schemaEntries, personLabel.String())
		personProperties := map[string]SchemaPropertyType{
			"name": {
				Indexed:   false,
				Array:     false,
				Existence: false,
				Type:      "STRING",
			},
			"age": {
				Indexed:   false,
				Array:     false,
				Existence: false,
				Type:      "INTEGER",
			},
		}
		assertSchemaEntryHasProperties(t, personEntry.Value.Properties, personProperties)

		companyEntry := getSchemaEntryByTypeOrLabel(schemaEntries, companyLabel.String())
		companyProperties := map[string]SchemaPropertyType{
			"name": {
				Indexed:   false,
				Array:     false,
				Existence: false,
				Type:      "STRING",
			},
			"founded": {
				Indexed:   false,
				Array:     false,
				Existence: false,
				Type:      "INTEGER",
			},
		}
		assertSchemaEntryHasProperties(t, companyEntry.Value.Properties, companyProperties)
	})

	// TODO keep extending the coverage of the schema tests:
	// - test different types such as float, duration etc ...
	// - test primitive array such as []float
	// - test Relationship
}

// assertSchemaHasLabel checks if the schema contains a node type with expected label
func assertSchemaHasLabel(t *testing.T, schemaEntries []SchemaEntry, label string) {
	foundLabel := slices.ContainsFunc(schemaEntries, func(schemaEntry SchemaEntry) bool {
		return schemaEntry.Key == label
	})

	if !foundLabel {
		t.Fatalf("label %s was not found in the schema", label)
	}
}

func getSchemaEntryByTypeOrLabel(schemaEntries []SchemaEntry, labelOrType string) SchemaEntry {
	idx := slices.IndexFunc(schemaEntries, func(schemaEntry SchemaEntry) bool {
		return schemaEntry.Key == labelOrType
	})

	return schemaEntries[idx]
}

func assertSchemaEntryHasProperties(t *testing.T, entryProperties map[string]SchemaPropertyType, expectedProperties map[string]SchemaPropertyType) {
	for name, expected := range expectedProperties {
		got, ok := entryProperties[name]
		if !ok {
			t.Fatalf("property %s expected for schema properties but not found, found properties: %v", name, entryProperties)
		}

		if got.Array != expected.Array {
			t.Fatalf("property %s not found: Array mismatch (expected=%t got=%t)", name, expected.Array, got.Array)
		}
		if got.Existence != expected.Existence {
			t.Fatalf("property %s not found: Existence mismatch (expected=%t got=%t)", name, expected.Existence, got.Existence)
		}
		if got.Indexed != expected.Indexed {
			t.Fatalf("property %s not found: Indexed mismatch (expected=%t got=%t)", name, expected.Indexed, got.Indexed)
		}
		if got.Type != expected.Type {
			t.Fatalf("property %s not found: Type mismatch (expected=%s got=%s)", name, expected.Type, got.Type)
		}

	}
}
