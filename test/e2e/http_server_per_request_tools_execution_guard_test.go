// Copyright (c) "Neo4j"
// Neo4j Sweden AB [http://neo4j.com]

//go:build e2e

package e2e

import (
	"context"
	"encoding/base64"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	mcpserver "github.com/neo4j/mcp/internal/server"

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
			extraHeaders: map[string]string{mcpserver.ReadOnlyHeader: "false"},
			toolName:     "write-cypher",
		},
		{
			name:         "When X-Neo4j-MCP-Tools contains the tool being called, the tool call should succeed",
			extraHeaders: map[string]string{mcpserver.ToolsHeader: "read-cypher, write-cypher"},
			toolName:     "write-cypher",
		},
		{
			name:         "When X-Neo4j-MCP-ReadOnly is true and the tool is read-only, the tool call should succeed",
			extraHeaders: map[string]string{mcpserver.ReadOnlyHeader: "true"},
			toolName:     "get-schema",
		},
		{
			name:         "When X-Neo4j-MCP-ReadOnly is true, a write tool call should be blocked",
			extraHeaders: map[string]string{mcpserver.ReadOnlyHeader: "true"},
			toolName:     "write-cypher",
			wantErr:      readOnlyError,
		},
		{
			name:         "When X-Neo4j-MCP-ReadOnly is true and the tool is not in X-Neo4j-MCP-Tools, the read-only error should take precedence",
			extraHeaders: map[string]string{mcpserver.ReadOnlyHeader: "true", mcpserver.ToolsHeader: "get-schema"},
			toolName:     "write-cypher",
			wantErr:      readOnlyError,
		},
		{
			name:         "When X-Neo4j-MCP-Tools is set and the tool is not in the list, the tool call should be blocked",
			extraHeaders: map[string]string{mcpserver.ToolsHeader: "get-schema, write-cypher"},
			toolName:     "read-cypher",
			wantErr:      toolError,
		},
		{
			name: "When X-Neo4j-MCP-ReadOnly is true and the tool is a read tool not present in X-Neo4j-MCP-Tools, the tool call should be blocked",
			extraHeaders: map[string]string{mcpserver.ReadOnlyHeader: "true",
				mcpserver.ToolsHeader: "get-schema, write-cypher"},
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
				"Authorization":     "Basic " + base64.StdEncoding.EncodeToString([]byte(cfg.Username+":"+cfg.Password)),
				mcpserver.URIHeader: cfg.URI,
			}
			for k, v := range tc.extraHeaders {
				headers[k] = v
			}

			session, err := newHTTPClient(t, ctx, baseURL+"/db/neo4j/mcp", headers)
			require.NoError(t, err, "expected initialize to succeed")
			defer session.Close()

			// get-schema takes no arguments; only cypher tools accept "query".
			arguments := map[string]any{}
			if tc.toolName != "get-schema" {
				arguments["query"] = "RETURN 1 AS n"
			}

			callToolResponse, err := session.CallTool(ctx, &mcp.CallToolParams{
				Name:      tc.toolName,
				Arguments: arguments,
			})
			require.NoError(t, err)

			if tc.wantErr != "" {
				textContent, ok := callToolResponse.Content[0].(*mcp.TextContent)

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
				"Authorization":     "Basic " + base64.StdEncoding.EncodeToString([]byte(cfg.Username+":"+cfg.Password)),
				mcpserver.URIHeader: cfg.URI,
			}

			session, err := newHTTPClient(t, ctx, baseURL+"/db/neo4j/mcp", headers)
			require.NoError(t, err, "expected initialize to succeed")
			defer session.Close()

			_, err = session.CallTool(ctx, &mcp.CallToolParams{
				Name: tc.toolName,
				Arguments: map[string]any{
					"query": "RETURN 1 AS n",
				},
			})
			require.ErrorContains(t, err, "unknown tool")
		})
	}
}
