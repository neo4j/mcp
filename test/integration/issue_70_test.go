//go:build integration

package integration

import (
	"context"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/neo4j/mcp/internal/tools"
	"github.com/neo4j/mcp/internal/tools/cypher"
	"github.com/neo4j/mcp/test/integration/helpers"
)

// https://github.com/neo4j/mcp/issues/70
func TestIssue70(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		handler func(deps *tools.ToolDependencies) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error)
	}{
		{
			name:    "read-cypher",
			handler: cypher.ReadCypherHandler,
		},
		{
			name:    "write-cypher",
			handler: cypher.ReadCypherHandler,
		},
	}

	for _, tt := range tests {

		t.Run(strings.Join([]string{tt.name, "should accept float parameter"}, " "), func(t *testing.T) {

			tc := helpers.NewTestContext(t, dbs.GetDriver())

			companyLabel := tc.GetUniqueLabel("Company")

			tc.SeedNode("Company", map[string]any{"prop": 1.2})
			tc.SeedNode("Company", map[string]any{"prop": 3.2})
			tc.SeedNode("Company", map[string]any{"prop": 4.2})

			handler := tt.handler(tc.Deps)
			handlerQuery := strings.Join(
				[]string{
					"MATCH (n:", companyLabel.String(), ")\n",
					"WHERE n.prop < $param1\n",
					"RETURN n\n",
				}, "")
			res := tc.CallTool(handler, map[string]any{
				"query": handlerQuery,
				"params": map[string]any{
					"param1": 3.5,
				},
			})

			var records []map[string]any
			tc.ParseJSONResponse(res, &records)

			if len(records) != 2 {
				t.Fatalf("expected 2 company, got %d", len(records))
			}
		})
		t.Run(strings.Join([]string{tt.name, "should accept integer parameter"}, " "), func(t *testing.T) {
			t.Skip("Skip issue 70 test until the underline issue is fixed")
			tc := helpers.NewTestContext(t, dbs.GetDriver())

			companyLabel := tc.GetUniqueLabel("Company")

			tc.SeedNode("Company", map[string]any{})
			tc.SeedNode("Company", map[string]any{})
			tc.SeedNode("Company", map[string]any{})
			tc.SeedNode("Company", map[string]any{})

			handler := tt.handler(tc.Deps)
			handlerQuery := strings.Join(
				[]string{
					"MATCH (n:", companyLabel.String(), ") RETURN n LIMIT $param1",
				}, "")
			res := tc.CallTool(handler, map[string]any{
				"query": handlerQuery,
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
			tc.AssertNodeHasLabel(company, companyLabel)
		})
	}
}
