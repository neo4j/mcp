// Copyright (c) "Neo4j"
// Neo4j Sweden AB [http://neo4j.com]

//go:build e2e

package e2e

import (
	"context"
	"testing"
	"time"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/neo4j/mcp/test/e2e/helpers"

	"github.com/stretchr/testify/require"
)

// TestStdioRequestTimeoutInitialize verifies that an expired server request
// timeout is surfaced as a clear error during initialize. STDIO has no per-request
// timeout header, so only the server maximum (--neo4j-request-timeout) applies.
func TestStdioRequestTimeoutInitialize(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		timeout string
		wantErr string
	}{
		{
			name:    "When the server request timeout expires, initialize should fail with the timeout error",
			timeout: "1ms",
			wantErr: "request timed out after 1ms",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			cfg := dbs.GetDriverConf()

			args := []string{
				"--neo4j-uri", cfg.URI,
				"--neo4j-username", cfg.Username,
				"--neo4j-password", cfg.Password,
				"--neo4j-database", cfg.Database,
				"--neo4j-telemetry", "false",
				"--neo4j-request-timeout", tc.timeout,
			}

			mcpClient, err := client.NewStdioMCPClient(server, []string{}, args...)
			require.NoError(t, err, "failed to create MCP client")
			defer mcpClient.Close()

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			_, err = mcpClient.Initialize(ctx, helpers.BuildInitializeRequest())
			require.ErrorContains(t, err, tc.wantErr)
		})
	}
}

// TestStdioRequestTimeoutToolCall verifies that a tool call exceeding the server
// request timeout is cancelled and reported as a tool error, while a call within
// budget succeeds.
func TestStdioRequestTimeoutToolCall(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		timeout   string
		query     string
		wantError bool
		wantMsg   string
	}{
		{
			name:      "When the tool call exceeds the request timeout, a timeout tool error should be returned",
			timeout:   "2s",
			query:     "CALL apoc.util.sleep(5000) RETURN 1 AS n",
			wantError: true,
			wantMsg:   "request timed out after 2s",
		},
		{
			name:      "When the tool call completes within the request timeout, it should succeed",
			timeout:   "5s",
			query:     "RETURN 1 AS n",
			wantError: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			cfg := dbs.GetDriverConf()

			args := []string{
				"--neo4j-uri", cfg.URI,
				"--neo4j-username", cfg.Username,
				"--neo4j-password", cfg.Password,
				"--neo4j-database", cfg.Database,
				"--neo4j-telemetry", "false",
				"--neo4j-request-timeout", tc.timeout,
			}

			mcpClient, err := client.NewStdioMCPClient(server, []string{}, args...)
			require.NoError(t, err, "failed to create MCP client")
			defer mcpClient.Close()

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			_, err = mcpClient.Initialize(ctx, helpers.BuildInitializeRequest())
			require.NoError(t, err, "expected initialize to succeed within the %s budget", tc.timeout)

			callToolResponse, err := mcpClient.CallTool(ctx, mcp.CallToolRequest{
				Params: mcp.CallToolParams{
					Name: "read-cypher",
					Arguments: map[string]any{
						"query": tc.query,
					},
				},
			})
			require.NoError(t, err)

			if tc.wantError {
				require.True(t, callToolResponse.IsError, "expected a tool error, got: %+v", callToolResponse)

				textContent, ok := callToolResponse.Content[0].(mcp.TextContent)
				require.True(t, ok)
				require.Contains(t, textContent.Text, tc.wantMsg)
				return
			}

			require.False(t, callToolResponse.IsError,
				"expected tool call to succeed, got: %+v", callToolResponse)
		})
	}
}
