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

func TestHTTPPerRequestToolsFilter(t *testing.T) {
	t.Parallel()

	baseURL := startHTTPModeServer(t)

	tests := []struct {
		name          string
		extraHeaders  map[string]string
		wantErr       bool
		wantToolNames []string
	}{
		{
			name:          "All tools returned when no filter header is set",
			wantToolNames: []string{"read-cypher", "write-cypher", "get-schema", "list-gds-procedures"},
		},
		{
			name:          "Only read-only tools when X-Neo4j-MCP-Readonly is true",
			extraHeaders:  map[string]string{"X-Neo4j-MCP-Readonly": "true"},
			wantToolNames: []string{"read-cypher", "get-schema", "list-gds-procedures"},
		},
		{
			name:          "All tools when X-Neo4j-MCP-Readonly is false",
			extraHeaders:  map[string]string{"X-Neo4j-MCP-Readonly": "false"},
			wantToolNames: []string{"read-cypher", "write-cypher", "get-schema", "list-gds-procedures"},
		},
		{
			name:          "All tools when X-Neo4j-MCP-Readonly is False (mixed case)",
			extraHeaders:  map[string]string{"X-Neo4j-MCP-Readonly": "False"},
			wantToolNames: []string{"read-cypher", "write-cypher", "get-schema", "list-gds-procedures"},
		},
		{
			name:         "Error when X-Neo4j-MCP-Readonly contains an invalid value",
			extraHeaders: map[string]string{"X-Neo4j-MCP-Readonly": "invalid"},
			wantErr:      true,
		},
		{
			name:          "Single tool filter via X-Neo4j-MCP-Tools",
			extraHeaders:  map[string]string{"X-Neo4j-MCP-Tools": "read-cypher"},
			wantToolNames: []string{"read-cypher"},
		},
		{
			name:          "Comma-separated tools via X-Neo4j-MCP-Tools",
			extraHeaders:  map[string]string{"X-Neo4j-MCP-Tools": "read-cypher, write-cypher"},
			wantToolNames: []string{"read-cypher", "write-cypher"},
		},
		{
			name:          "All tools when all available tools are listed in X-Neo4j-MCP-Tools",
			extraHeaders:  map[string]string{"X-Neo4j-MCP-Tools": "read-cypher, write-cypher, get-schema, list-gds-procedures"},
			wantToolNames: []string{"read-cypher", "write-cypher", "get-schema", "list-gds-procedures"},
		},
		{
			name:         "Error when X-Neo4j-MCP-Tools contains an invalid tool name",
			extraHeaders: map[string]string{"X-Neo4j-MCP-Tools": "batman-tool"},
			wantErr:      true,
		},
		{
			name:         "Error when X-Neo4j-MCP-Tools contains a mixed valid tool names",
			extraHeaders: map[string]string{"X-Neo4j-MCP-Tools": "read-cypher, batman-tool"},
			wantErr:      true,
		},
		{
			name:         "Error when X-Neo4j-MCP-Tools contains and \"\" string",
			extraHeaders: map[string]string{"X-Neo4j-MCP-Tools": ""},
			wantErr:      true,
		},
		{
			name: "X-Neo4j-MCP-Tools and X-Neo4j-MCP-Readonly applied as intersection",
			extraHeaders: map[string]string{
				"X-Neo4j-MCP-Tools":    "read-cypher, write-cypher",
				"X-Neo4j-MCP-Readonly": "true",
			},
			// write-cypher is not read-only, so it is excluded despite being in the tools list
			wantToolNames: []string{"read-cypher"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mcpURL := baseURL + "/db/neo4j/mcp"
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

			httpClient := newHTTPClient(t, mcpURL, headers)
			defer httpClient.Close()

			require.NoError(t, httpClient.Start(ctx), "http client failed to start")

			_, err := httpClient.Initialize(ctx, helpers.BuildInitializeRequest())
			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err, "expected initialize to succeed")

			listToolsResponse, err := httpClient.ListTools(ctx, mcp.ListToolsRequest{})
			require.NoError(t, err, "failed to list tools")

			toolNames := make([]string, len(listToolsResponse.Tools))
			for i, tool := range listToolsResponse.Tools {
				toolNames[i] = tool.Name
			}
			require.ElementsMatch(t, tc.wantToolNames, toolNames)
		})
	}
}
