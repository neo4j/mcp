// Copyright (c) "Neo4j"
// Neo4j Sweden AB [http://neo4j.com]

//go:build integration

package integration

import (
	"context"
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
		res := helpers.CallTool(tc, read, cypher.ReadCypherInput{
			Query:  "MATCH (p:" + personLabel.String() + " {name: $name}) RETURN p",
			Params: cypher.Params{"name": "Alice"},
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
		textError := helpers.GetToolError(tc, read, cypher.ReadCypherInput{
			Query:  "CREATE (p:" + personLabel.String() + ") SET p.name = $name RETURN p",
			Params: cypher.Params{"name": "Alice"},
		})

		if !strings.Contains(textError, "read-cypher can only run read-only Cypher statements.") {
			t.Fatal("read-cypher no rejected CREATE query")
		}
	})

	t.Run("read-cypher should omit stored embeddings and keep other long lists", func(t *testing.T) {
		tc := helpers.NewTestContext(t, dbs.GetDriver())

		embedding := make([]float64, 1536)
		for i := range embedding {
			embedding[i] = 0.25
		}
		movieLabel, err := tc.SeedNode("Movie", map[string]any{"title": "Heat", "embedding": embedding})
		if err != nil {
			t.Fatalf("failed to seed data: %v", err)
		}

		read := cypher.ReadCypherHandler(tc.Deps)
		res := tc.CallTool(read, map[string]any{
			"query": "MATCH (m:" + movieLabel + ") RETURN m, m.embedding AS e, range(1, 300) AS ids",
		})

		var records []map[string]any
		tc.ParseJSONResponse(res, &records)
		if len(records) != 1 {
			t.Fatalf("expected 1 record, got %d", len(records))
		}

		node, ok := records[0]["m"].(map[string]any)
		if !ok {
			t.Fatalf("expected m to be map[string]any, got %T", records[0]["m"])
		}
		tc.AssertNodeProperties(node, map[string]any{
			"title":     "Heat",
			"embedding": "<embedding: 1536 values omitted>",
		})
		if records[0]["e"] != "<embedding: 1536 values omitted>" {
			t.Errorf("e = %v, want the embedding placeholder", records[0]["e"])
		}
		if ids, ok := records[0]["ids"].([]any); !ok || len(ids) != 300 {
			t.Errorf("ids = %v, want the 300 integers from range(1, 300)", records[0]["ids"])
		}
	})

	t.Run("read-cypher should omit native VECTOR values", func(t *testing.T) {
		tc := helpers.NewTestContext(t, dbs.GetDriver())

		// The VECTOR type is not available on older servers such as the default test image;
		// run with NEO4J_IMAGE=neo4j:community to exercise it.
		if _, err := tc.Service.ExecuteReadQuery(context.Background(), "RETURN vector([1.0], 1, FLOAT32) AS v", nil); err != nil {
			t.Skipf("server does not support the VECTOR type: %v", err)
		}

		embedding := make([]float64, 1536)
		read := cypher.ReadCypherHandler(tc.Deps)
		res := tc.CallTool(read, map[string]any{
			"query":  "RETURN vector($embedding, 1536, FLOAT32) AS v",
			"params": map[string]any{"embedding": embedding},
		})

		var records []map[string]any
		tc.ParseJSONResponse(res, &records)
		if len(records) != 1 {
			t.Fatalf("expected 1 record, got %d", len(records))
		}
		if records[0]["v"] != "<embedding: 1536 values omitted>" {
			t.Errorf("v = %v, want the embedding placeholder", records[0]["v"])
		}
	})
}
