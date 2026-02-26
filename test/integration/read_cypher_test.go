// Copyright (c) "Neo4j"
// Neo4j Sweden AB [http://neo4j.com]

//go:build integration

package integration

import (
	"strings"
	"testing"

	"github.com/neo4j/mcp/internal/tools/cypher"
	"github.com/neo4j/mcp/test/integration/helpers"
)

func TestReadCypher(t *testing.T) {
	t.Parallel()
	t.Run("read-cypher should able to read from a neo4j instance", func(t *testing.T) {
		tc := helpers.NewTestContext(t, dbs.GetDriver())

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
		tc.AssertNodeProperties(pNode, map[string]any{"name": "Alice"})
		tc.AssertNodeHasLabel(pNode, personLabel)
	})

	t.Run("read-cypher should not be able to perform state mutation", func(t *testing.T) {
		tc := helpers.NewTestContext(t, dbs.GetDriver())

		personLabel := tc.GetUniqueLabel("Person")

		read := cypher.ReadCypherHandler(tc.Deps)
		textError := tc.GetToolError(read, map[string]any{
			"query":  "CREATE (p:" + personLabel + ") SET p.name = $name RETURN p",
			"params": map[string]any{"name": "Alice"},
		})

		if !strings.Contains(textError, "read-cypher can only run read-only Cypher statements.") {
			t.Fatal("read-cypher no rejected CREATE query")
		}
	})

}
