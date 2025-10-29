//go:build integration

package integration

import (
	"strings"
	"testing"

	"github.com/neo4j/mcp/internal/tools/cypher"
	"github.com/neo4j/mcp/test/integration/helpers"
)

// https://github.com/neo4j/mcp/issues/70
func TestIssue70_Int(t *testing.T) {
	t.Parallel()
	//t.Skip("Skip issue 70 until integer support as parameter is supported")
	tc := helpers.NewTestContext(t, dbs.GetDriver())

	companyLabel := tc.GetUniqueLabel("Company")

	tc.SeedNode("Company", map[string]any{})
	tc.SeedNode("Company", map[string]any{})
	tc.SeedNode("Company", map[string]any{})
	tc.SeedNode("Company", map[string]any{})

	read := cypher.ReadCypherHandler(tc.Deps)
	readQuery := strings.Join(
		[]string{
			"MATCH (n:", companyLabel.String(), ") RETURN n LIMIT $param1",
		}, "")
	res := tc.CallTool(read, map[string]any{
		"query": readQuery,
		"params": map[string]int{
			"param1": 1,
		},
	})

	var records []map[string]any
	tc.ParseJSONResponse(res, &records)

	if len(records) != 1 {
		t.Fatalf("expected 1 company, got %d", len(records))
	}

	company := records[0]["n"].(map[string]any)
	helpers.AssertNodeHasLabel(t, company, companyLabel)
}

// https://github.com/neo4j/mcp/issues/70
func TestIssue70_Float(t *testing.T) {
	t.Parallel()
	tc := helpers.NewTestContext(t, dbs.GetDriver())

	companyLabel := tc.GetUniqueLabel("Company")

	tc.SeedNode("Company", map[string]any{"prop": 1.2})
	tc.SeedNode("Company", map[string]any{"prop": 3.2})
	tc.SeedNode("Company", map[string]any{"prop": 4.2})

	read := cypher.ReadCypherHandler(tc.Deps)
	readQuery := strings.Join(
		[]string{
			"MATCH (n:", companyLabel.String(), ")\n",
			"WHERE n.prop < $param1\n",
			"RETURN n\n",
		}, "")
	res := tc.CallTool(read, map[string]any{
		"query": readQuery,
		"params": map[string]any{
			"param1": 3.5,
		},
	})

	var records []map[string]any
	tc.ParseJSONResponse(res, &records)

	if len(records) != 2 {
		t.Fatalf("expected 2 company, got %d", len(records))
	}
}
