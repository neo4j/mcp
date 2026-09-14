// Copyright (c) "Neo4j"
// Neo4j Sweden AB [http://neo4j.com]

//go:build integration

package integration

import (
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/neo4j/mcp/internal/tools"
	"github.com/neo4j/mcp/internal/tools/cypher"
	"github.com/neo4j/mcp/test/integration/helpers"
)

// https://github.com/neo4j/mcp/issues/70
func TestIssue70(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		callTool func(tc *helpers.TestContext, deps *tools.ToolDependencies, query string, params cypher.Params) *mcp.CallToolResult
	}{
		{
			name: "read-cypher",
			callTool: func(tc *helpers.TestContext, deps *tools.ToolDependencies, query string, params cypher.Params) *mcp.CallToolResult {
				return helpers.CallTool(tc, cypher.ReadCypherHandler(deps), cypher.ReadCypherInput{Query: query, Params: params})
			},
		},
		{
			name: "write-cypher",
			callTool: func(tc *helpers.TestContext, deps *tools.ToolDependencies, query string, params cypher.Params) *mcp.CallToolResult {
				return helpers.CallTool(tc, cypher.WriteCypherHandler(deps), cypher.WriteCypherInput{Query: query, Params: params})
			},
		},
	}

	for _, tt := range tests {

		t.Run(strings.Join([]string{tt.name, "should accept float parameter"}, " "), func(t *testing.T) {

			tc := helpers.NewTestContext(t, dbs.GetDriver())

			companyLabel := tc.GetUniqueLabel("Company")

			_, err := tc.SeedNode("Company", map[string]any{"prop": 1.2})
			if err != nil {
				t.Fatalf("failed to seed Company node: %v", err)
			}
			_, err = tc.SeedNode("Company", map[string]any{"prop": 3.2})
			if err != nil {
				t.Fatalf("failed to seed Company node: %v", err)
			}
			_, err = tc.SeedNode("Company", map[string]any{"prop": 4.2})
			if err != nil {
				t.Fatalf("failed to seed Company node: %v", err)
			}

			handlerQuery := strings.Join(
				[]string{
					"MATCH (n:", companyLabel.String(), ")\n",
					"WHERE n.prop < $param1\n",
					"RETURN n\n",
				}, "")
			res := tt.callTool(tc, tc.Deps, handlerQuery, cypher.Params{
				"param1": 3.5,
			})

			var records []map[string]any
			tc.ParseJSONResponse(res, &records)

			if len(records) != 2 {
				t.Fatalf("expected 2 company, got %d", len(records))
			}
		})
		t.Run(strings.Join([]string{tt.name, "should accept integer parameter"}, " "), func(t *testing.T) {
			tc := helpers.NewTestContext(t, dbs.GetDriver())

			companyLabel := tc.GetUniqueLabel("Company")

			_, err := tc.SeedNode("Company", map[string]any{})
			if err != nil {
				t.Fatalf("failed to seed Company node: %v", err)
			}
			_, err = tc.SeedNode("Company", map[string]any{})
			if err != nil {
				t.Fatalf("failed to seed Company node: %v", err)
			}
			_, err = tc.SeedNode("Company", map[string]any{})
			if err != nil {
				t.Fatalf("failed to seed Company node: %v", err)
			}
			_, err = tc.SeedNode("Company", map[string]any{})
			if err != nil {
				t.Fatalf("failed to seed Company node: %v", err)
			}

			handlerQuery := strings.Join(
				[]string{
					"MATCH (n:", companyLabel.String(), ") RETURN n LIMIT $param1",
				}, "")
			res := tt.callTool(tc, tc.Deps, handlerQuery, cypher.Params{
				"param1": 1,
			})

			var records []map[string]any
			tc.ParseJSONResponse(res, &records)

			if len(records) != 1 {
				t.Fatalf("expected 1 company, got %d", len(records))
			}

			company := records[0]["n"].(map[string]any)
			tc.AssertNodeHasLabel(company, companyLabel)
		})
	}
}
