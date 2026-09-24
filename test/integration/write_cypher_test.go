// Copyright (c) "Neo4j"
// Neo4j Sweden AB [http://neo4j.com]

//go:build integration

package integration

import (
	"testing"

	"github.com/neo4j/mcp/internal/tools/cypher"
	"github.com/neo4j/mcp/test/integration/helpers"
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
