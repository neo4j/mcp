// Copyright (c) "Neo4j"
// Neo4j Sweden AB [http://neo4j.com]

//go:build e2e

package e2e

import (
	"context"
	"testing"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/neo4j/mcp/test/e2e/helpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEmptyParamsE2E(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	cfg := dbs.GetDriverConf()
	args := []string{
		"--neo4j-uri", cfg.URI,
		"--neo4j-username", cfg.Username,
		"--neo4j-password", cfg.Password,
		"--neo4j-database", cfg.Database,
	}

	mcpClient, err := client.NewStdioMCPClient(server, []string{}, args...)
	require.NoError(t, err, "failed to create MCP client")
	t.Cleanup(func() {
		mcpClient.Close()
	})

	_, err = mcpClient.Initialize(ctx, helpers.BuildInitializeRequest())
	require.NoError(t, err, "failed to initialize MCP server")

	t.Run("write-cypher succeeds without params argument", func(t *testing.T) {
		t.Parallel()
		helpers.NewE2ETestContext(t, dbs.GetDriver())
		tc := helpers.NewE2ETestContext(t, dbs.GetDriver())
		label := tc.GetUniqueLabel("NoParams")
		// Call write-cypher with only the required `query` field — no `params`.
		// This verifies that `params` is truly optional.
		resp, err := mcpClient.CallTool(ctx, mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Name: "write-cypher",
				Arguments: map[string]any{
					"query": "CREATE (n:" + label.String() + " ) SET n.prop = \"test\" RETURN n",
				},
			},
		})
		require.NoError(t, err, "CallTool returned an unexpected transport error")
		require.False(t, resp.IsError, "write-cypher failed: %v", resp.Content)

		textContent, ok := mcp.AsTextContent(resp.Content[0])
		require.True(t, ok, "expected TextContent in response")
		assert.NotEmpty(t, textContent.Text, "expected non-empty response body")
	})

	t.Run("read-cypher succeeds without params argument", func(t *testing.T) {
		t.Parallel()
		helpers.NewE2ETestContext(t, dbs.GetDriver())

		// Call read-cypher with only the required `query` field — no `params`.
		// This verifies that `params` is truly optional.
		resp, err := mcpClient.CallTool(ctx, mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Name: "read-cypher",
				Arguments: map[string]any{
					"query": "RETURN 1",
				},
			},
		})
		require.NoError(t, err, "CallTool returned an unexpected transport error")
		require.False(t, resp.IsError, "read-cypher failed: %v", resp.Content)

		textContent, ok := mcp.AsTextContent(resp.Content[0])
		require.True(t, ok, "expected TextContent in response")
		assert.NotEmpty(t, textContent.Text, "expected non-empty response body")
	})
}
