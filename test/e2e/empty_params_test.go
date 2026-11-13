// Copyright (c) "Neo4j"
// Neo4j Sweden AB [http://neo4j.com]

//go:build e2e

package e2e

import (
	"context"
	"os/exec"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/neo4j/mcp/test/e2e/helpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEmptyParamsE2E(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	cfg := dbs.GetDriverConf()
	args := []string{
		"--uri", cfg.URI,
		"--username", cfg.Username,
		"--password", cfg.Password,
		"--database", cfg.Database,
	}

	mcpClient := helpers.NewTestClient()
	session, err := mcpClient.Connect(ctx, &mcp.CommandTransport{Command: exec.Command(server, args...)}, nil)
	require.NoError(t, err, "failed to connect MCP client")
	t.Cleanup(func() {
		session.Close()
	})

	t.Run("write-cypher succeeds without params argument", func(t *testing.T) {
		t.Parallel()
		helpers.NewE2ETestContext(t, dbs.GetDriver())
		tc := helpers.NewE2ETestContext(t, dbs.GetDriver())
		label := tc.GetUniqueLabel("NoParams")
		// Call write-cypher with only the required `query` field — no `params`.
		// This verifies that `params` is truly optional.
		resp, err := session.CallTool(ctx, &mcp.CallToolParams{
			Name: "write-cypher",
			Arguments: map[string]any{
				"query": "CREATE (n:" + label.String() + " ) SET n.prop = \"test\" RETURN n",
			},
		})
		require.NoError(t, err, "CallTool returned an unexpected transport error")
		require.False(t, resp.IsError, "write-cypher failed: %v", resp.Content)

		textContent, ok := resp.Content[0].(*mcp.TextContent)
		require.True(t, ok, "expected TextContent in response")
		assert.NotEmpty(t, textContent.Text, "expected non-empty response body")
	})

	t.Run("read-cypher succeeds without params argument", func(t *testing.T) {
		t.Parallel()
		helpers.NewE2ETestContext(t, dbs.GetDriver())

		// Call read-cypher with only the required `query` field — no `params`.
		// This verifies that `params` is truly optional.
		resp, err := session.CallTool(ctx, &mcp.CallToolParams{
			Name: "read-cypher",
			Arguments: map[string]any{
				"query": "RETURN 1",
			},
		})
		require.NoError(t, err, "CallTool returned an unexpected transport error")
		require.False(t, resp.IsError, "read-cypher failed: %v", resp.Content)

		textContent, ok := resp.Content[0].(*mcp.TextContent)
		require.True(t, ok, "expected TextContent in response")
		assert.NotEmpty(t, textContent.Text, "expected non-empty response body")
	})
}
