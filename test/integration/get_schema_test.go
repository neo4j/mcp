// Copyright (c) "Neo4j"
// Neo4j Sweden AB [http://neo4j.com]

//go:build integration

package integration

import (
	"context"
	"slices"
	"strings"
	"testing"

	"github.com/neo4j/mcp/internal/tools/cypher"
	"github.com/neo4j/mcp/test/integration/helpers"
)

type SchemaItem struct {
	Key   string       `json:"key"`
	Value SchemaDetail `json:"value"`
}

type SchemaDetail struct {
	Type          string                  `json:"type"`
	Properties    map[string]string       `json:"properties,omitempty"`
	Relationships map[string]Relationship `json:"relationships,omitempty"`
}

type Relationship struct {
	Direction  string            `json:"direction"`
	Labels     []string          `json:"labels"` // List of target node labels
	Properties map[string]string `json:"properties,omitempty"`
}

func TestGetSchema(t *testing.T) {
	t.Parallel()
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

	getSchema := cypher.GetSchemaHandler(tc.Deps, 100)
	res := tc.CallTool(getSchema, nil)

	var schemaEntries []SchemaItem
	tc.ParseJSONResponse(res, &schemaEntries)

	if len(schemaEntries) == 0 {
		t.Fatal("expected schema to contain at least one entry")
	}
	assertSchemaHasLabel(t, schemaEntries, personLabel.String())
	assertSchemaHasLabel(t, schemaEntries, companyLabel.String())

	personEntry := getSchemaItemByTypeOrLabel(schemaEntries, personLabel.String())
	personProperties := map[string]any{
		"name": "STRING",
		"age":  "INTEGER",
	}
	assertSchemaEntryHasProperties(t, personEntry.Value.Properties, personProperties)

	companyEntry := getSchemaItemByTypeOrLabel(schemaEntries, companyLabel.String())
	companyProperties := map[string]any{
		"name":    "STRING",
		"founded": "INTEGER",
	}
	assertSchemaEntryHasProperties(t, companyEntry.Value.Properties, companyProperties)
}

func TestGetSchemaExcludesBloomMetadata(t *testing.T) {
	t.Parallel()
	tc := helpers.NewTestContext(t, dbs.GetDriver())

	movieLabel, err := tc.SeedNode("Movie", map[string]any{"title": "The Matrix"})
	if err != nil {
		t.Fatalf("failed to seed Movie node: %v", err)
	}

	// Bloom stores perspectives and scenes under fixed labels, so they cannot be made unique per test.
	// Tag them with the test ID and remove them explicitly.
	bloomParams := map[string]any{"testId": tc.TestID}
	_, err = tc.Service.ExecuteWriteQuery(context.Background(),
		"CREATE (:_Bloom_Perspective_ {name: 'Movies', testId: $testId})-[:_Bloom_HAS_SCENE_]->(:_Bloom_Scene_ {name: 'Scene 1', testId: $testId})",
		bloomParams)
	if err != nil {
		t.Fatalf("failed to seed Bloom metadata: %v", err)
	}
	t.Cleanup(func() {
		if _, err := tc.Service.ExecuteWriteQuery(context.Background(),
			"MATCH (n) WHERE (n:_Bloom_Perspective_ OR n:_Bloom_Scene_) AND n.testId = $testId DETACH DELETE n",
			bloomParams); err != nil {
			t.Logf("cleanup of Bloom metadata failed: %v", err)
		}
	})

	getSchema := cypher.GetSchemaHandler(tc.Deps, 100)
	res := tc.CallTool(getSchema, nil)

	var schemaEntries []SchemaItem
	tc.ParseJSONResponse(res, &schemaEntries)

	assertSchemaHasLabel(t, schemaEntries, movieLabel.String())
	for _, entry := range schemaEntries {
		if strings.HasPrefix(entry.Key, "_Bloom_") {
			t.Errorf("expected Bloom metadata to be excluded from the schema, found %s", entry.Key)
		}
	}
}

// assertSchemaHasLabel checks if the schema contains a node type with expected label
func assertSchemaHasLabel(t *testing.T, schemaEntries []SchemaItem, label string) {
	foundLabel := slices.ContainsFunc(schemaEntries, func(schemaEntry SchemaItem) bool {
		return schemaEntry.Key == label
	})

	if !foundLabel {
		t.Fatalf("label %s was not found in the schema", label)
	}
}

func getSchemaItemByTypeOrLabel(schemaEntries []SchemaItem, labelOrType string) SchemaItem {
	idx := slices.IndexFunc(schemaEntries, func(schemaEntry SchemaItem) bool {
		return schemaEntry.Key == labelOrType
	})

	return schemaEntries[idx]
}

func assertSchemaEntryHasProperties(t *testing.T, entryProperties map[string]string, expectedProperties map[string]any) {
	for name, expected := range expectedProperties {
		got, ok := entryProperties[name]
		if !ok {
			t.Fatalf("property %s expected for schema properties but not found, found properties: %v", name, entryProperties)
		}

		if got != expected {
			t.Fatalf("property with name: %s has an invalid type (expected=%v got=%v)", name, expected, got)
		}
	}
}
