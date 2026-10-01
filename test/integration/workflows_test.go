// Copyright (c) "Neo4j"
// Neo4j Sweden AB [http://neo4j.com]

//go:build integration

package integration

import (
	"testing"

	"github.com/neo4j/mcp/internal/tools/cypher"
	"github.com/neo4j/mcp/test/integration/helpers"
)

func TestWriteThenRead(t *testing.T) {
	t.Parallel()
	tc := helpers.NewTestContext(t, dbs.GetDriver())

	companyLabel := tc.GetUniqueLabel("Company")

	write := cypher.WriteCypherHandler(tc.Deps)
	helpers.CallTool(tc, write, cypher.WriteCypherInput{
		Query:  "CREATE (c:" + companyLabel.String() + " {name: $name, industry: $industry}) RETURN c",
		Params: cypher.Params{"name": "Neo4j", "industry": "Database"},
	})

	read := cypher.ReadCypherHandler(tc.Deps)
	res := helpers.CallTool(tc, read, cypher.ReadCypherInput{
		Query:  "MATCH (c:" + companyLabel.String() + ") RETURN c",
		Params: cypher.Params{},
	})

	var records []map[string]any
	tc.ParseJSONResponse(res, &records)

	if len(records) != 1 {
		t.Fatalf("expected 1 company, got %d", len(records))
	}

	company := records[0]["c"].(map[string]any)
	tc.AssertNodeProperties(company, map[string]any{
		"name":     "Neo4j",
		"industry": "Database",
	})
	tc.AssertNodeHasLabel(company, companyLabel)
}
