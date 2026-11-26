// Copyright (c) "Neo4j"
// Neo4j Sweden AB [http://neo4j.com]

//go:build integration

package integration

import (
	"testing"

	"github.com/neo4j/mcp/internal/tools/cypher"
	"github.com/neo4j/mcp/test/integration/helpers"
	"github.com/neo4j/neo4j-go-driver/v6/neo4j"
)

func TestWriteCypher(t *testing.T) {
	t.Parallel()
	tc := helpers.NewTestContext(t, dbs.GetDriver())

	personLabel := tc.GetUniqueLabel("Person")

	write := cypher.WriteCypherHandler(tc.Deps)
	helpers.CallTool(tc, write, cypher.WriteCypherInput{
		Query:  "CREATE (p:" + personLabel.String() + " {name: $name}) RETURN p",
		Params: cypher.Params{"name": "Alice"},
	})

	tc.VerifyNodeInDB(personLabel, map[string]any{"name": "Alice"})
}

func TestWriteCypherOmitEmbeddings(t *testing.T) {
	t.Parallel()
	tc := helpers.NewTestContext(t, dbs.GetDriver())

	movieLabel := tc.GetUniqueLabel("Movie")

	write := cypher.WriteCypherHandler(tc.Deps)
	res := helpers.CallTool(tc, write, cypher.WriteCypherInput{
		Query:          "CREATE (m:" + movieLabel.String() + " {title: $title, embedding: $embedding}) RETURN m",
		Params:         cypher.Params{"title": "Heat", "embedding": testEmbedding(1536)},
		OmitEmbeddings: true,
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

	// Only the response is sanitized; the stored embedding is untouched.
	record := tc.VerifyNodeInDB(movieLabel, map[string]any{"title": "Heat"})
	stored, _ := record.Values[0].(neo4j.Node)
	if e, ok := stored.Props["embedding"].([]any); !ok || len(e) != 1536 {
		t.Errorf("stored embedding = %v, want the list of 1536 floats", stored.Props["embedding"])
	}
}
