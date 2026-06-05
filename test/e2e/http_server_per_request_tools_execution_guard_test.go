// Copyright (c) "Neo4j"
// Neo4j Sweden AB [http://neo4j.com]

//go:build e2e

package e2e

import (
	"context"
	"encoding/base64"
	"testing"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/neo4j/mcp/test/e2e/helpers"

	"github.com/stretchr/testify/require"
)

func TestHTTPPerRequestToolsExecutionGuard(t *testing.T) {
	t.Parallel()

	baseURL := startHTTPModeServer(t)

	readOnlyError := "is not permitted in read-only mode"
	toolError := "is not in the list of configured tools"

	tests := []struct {
		name         string
		extraHeaders map[string]string
		wantErr      string
		toolName     string
	}{
		{
			name:     "When neither X-Neo4j-MCP-Tools nor X-Neo4j-MCP-ReadOnly are set, the tool call should succeed",
			toolName: "write-cypher",
		},
		{
			name:         "When X-Neo4j-MCP-ReadOnly is false, the tool call should succeed",
			extraHeaders: map[string]string{"X-Neo4j-MCP-ReadOnly": "false"},
			toolName:     "write-cypher",
		},
		{
			name:         "When X-Neo4j-MCP-Tools contains the tool being called, the tool call should succeed",
			extraHeaders: map[string]string{"X-Neo4j-MCP-Tools": "read-cypher, write-cypher"},
			toolName:     "write-cypher",
		},
		{
			name:         "When X-Neo4j-MCP-ReadOnly is true and the tool is read-only, the tool call should succeed",
			extraHeaders: map[string]string{"X-Neo4j-MCP-ReadOnly": "true"},
			toolName:     "get-schema",
		},
		{
			name:         "When X-Neo4j-MCP-ReadOnly is true, a write tool call should be blocked",
			extraHeaders: map[string]string{"X-Neo4j-MCP-ReadOnly": "true"},
			toolName:     "write-cypher",
			wantErr:      readOnlyError,
		},
		{
			name:         "When X-Neo4j-MCP-ReadOnly is true and the tool is not in X-Neo4j-MCP-Tools, the read-only error should take precedence",
			extraHeaders: map[string]string{"X-Neo4j-MCP-ReadOnly": "true", "X-Neo4j-MCP-Tools": "get-schema"},
			toolName:     "write-cypher",
			wantErr:      readOnlyError,
		},
		{
			name:         "When X-Neo4j-MCP-Tools is set and the tool is not in the list, the tool call should be blocked",
			extraHeaders: map[string]string{"X-Neo4j-MCP-Tools": "get-schema, write-cypher"},
			toolName:     "read-cypher",
			wantErr:      toolError,
		},
		{
			name: "When X-Neo4j-MCP-ReadOnly is true and the tool is a read tool not present in X-Neo4j-MCP-Tools, the tool call should be blocked",
			extraHeaders: map[string]string{"X-Neo4j-MCP-ReadOnly": "true",
				"X-Neo4j-MCP-Tools": "get-schema, write-cypher"},
			toolName: "read-cypher",
			wantErr:  toolError,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			cfg := dbs.GetDriverConf()

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			headers := map[string]string{
				"Authorization":   "Basic " + base64.StdEncoding.EncodeToString([]byte(cfg.Username+":"+cfg.Password)),
				"X-Neo4j-MCP-URI": cfg.URI,
			}
			for k, v := range tc.extraHeaders {
				headers[k] = v
			}

			httpClient := newHTTPClient(t, baseURL+"/db/neo4j/mcp", headers)

			defer httpClient.Close()

			require.NoError(t, httpClient.Start(ctx), "http client failed to start")

			_, err := httpClient.Initialize(ctx, helpers.BuildInitializeRequest())
			require.NoError(t, err, "expected initialize to succeed")

			callToolResponse, err := httpClient.CallTool(ctx, mcp.CallToolRequest{
				Params: mcp.CallToolParams{
					Name: tc.toolName,
					Arguments: map[string]any{
						"query": "RETURN 1 AS n",
					},
				},
			})
			require.NoError(t, err)

			if tc.wantErr != "" {
				textContent, ok := callToolResponse.Content[0].(mcp.TextContent)

				require.True(t, ok)
				require.Contains(t, textContent.Text, tc.wantErr)
			} else {
				require.False(t, callToolResponse.IsError,
					"expected tool call to be allowed, got: %+v", callToolResponse)
			}

		})
	}
}

// This test was added as we're reliant on the tools execution guard middleware not being invoked in the case of invalid tool names, or tool names with different casing.
// This is true because we rely on the MCP SDK returning an error before executing the middleware, in these cases.
func TestHTTPPerRequestToolsExecutionGuardInvalidTool(t *testing.T) {
	t.Parallel()

	baseURL := startHTTPModeServer(t)

	tests := []struct {
		name     string
		toolName string
	}{
		{
			name:     "When invalid tool is called, tool handler middleware should not be invoked",
			toolName: "invalid-tool",
		},
		{
			name:     "When valid tool with invalid casing is called, tool handler middleware should not be invoked",
			toolName: "Read-cypher",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			cfg := dbs.GetDriverConf()

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			headers := map[string]string{
				"Authorization":   "Basic " + base64.StdEncoding.EncodeToString([]byte(cfg.Username+":"+cfg.Password)),
				"X-Neo4j-MCP-URI": cfg.URI,
			}

			httpClient := newHTTPClient(t, baseURL+"/db/neo4j/mcp", headers)

			defer httpClient.Close()

			require.NoError(t, httpClient.Start(ctx), "http client failed to start")

			_, err := httpClient.Initialize(ctx, helpers.BuildInitializeRequest())
			require.NoError(t, err, "expected initialize to succeed")

			_, err = httpClient.CallTool(ctx, mcp.CallToolRequest{
				Params: mcp.CallToolParams{
					Name: tc.toolName,
					Arguments: map[string]any{
						"query": "RETURN 1 AS n",
					},
				},
			})
			require.ErrorContains(t, err, "invalid params: tool")
		})
	}
}
