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

func TestMalformedArgumentsE2E(t *testing.T) {
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

	tests := []struct {
		name            string
		arguments       any
		wantTextContain string
	}{
		{
			name:            "arguments is a string instead of an object",
			arguments:       "invalid string instead of map",
			wantTextContain: "arguments",
		},
		{
			name:            "required query field is missing",
			arguments:       map[string]any{},
			wantTextContain: "query",
		},
		{
			name:            "query field has the wrong type",
			arguments:       map[string]any{"query": 123},
			wantTextContain: "query",
		},
		{
			name: "unexpected additional properties",
			arguments: map[string]any{
				"query":         "MATCH (n) RETURN n",
				"invalid_field": "value",
			},
			wantTextContain: "invalid_field",
		},
	}

	for _, toolName := range []string{"read-cypher", "write-cypher"} {
		t.Run(toolName, func(t *testing.T) {
			for _, tc := range tests {
				t.Run(tc.name, func(t *testing.T) {
					resp, err := session.CallTool(ctx, &mcp.CallToolParams{
						Name:      toolName,
						Arguments: tc.arguments,
					})
					require.NoError(t, err, "CallTool returned an unexpected transport error")
					require.True(t, resp.IsError, "expected malformed arguments to be rejected as a tool error")

					textContent, ok := resp.Content[0].(*mcp.TextContent)
					require.True(t, ok, "expected TextContent in response")
					assert.Contains(t, textContent.Text, `validating "arguments"`, "expected the rejection to come from schema validation")
					assert.Contains(t, textContent.Text, tc.wantTextContain)
				})
			}
		})
	}
}
