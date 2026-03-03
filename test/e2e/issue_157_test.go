// Copyright (c) "Neo4j"
// Neo4j Sweden AB [http://neo4j.com]

//go:build e2e

package e2e

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/neo4j/mcp/test/e2e/helpers"
	"github.com/stretchr/testify/require"
)

// test for issue https://github.com/neo4j/mcp/issues/157

type toolInputSchema struct {
	Properties map[string]interface{} `json:"properties"`
}

type toolInfo struct {
	Name        string          `json:"name"`
	InputSchema toolInputSchema `json:"inputSchema"`
}

type listToolsResponse struct {
	Tools []toolInfo `json:"tools"`
}

func TestIssue157(t *testing.T) {
	t.Parallel()
	// Create MCP client
	ctx := context.Background()

	cfg := dbs.GetDriverConf()
	args := []string{
		"--neo4j-uri", cfg.URI,
		"--neo4j-username", cfg.Username,
		"--neo4j-password", cfg.Password,
		"--neo4j-database", cfg.Database,
	}

	mcpClient, err := client.NewStdioMCPClient(server, []string{}, args...)
	if err != nil {
		t.Fatalf("failed to create MCP client: %v", err)
	}

	// Initialize the server
	_, err = mcpClient.Initialize(ctx, helpers.BuildInitializeRequest())
	if err != nil {
		t.Fatalf("failed to initialize MCP server: %v", err)
	}
	t.Cleanup(func() {
		mcpClient.Close()
	})

	t.Run("all tools returned from listTools should contains inputSchema properties", func(t *testing.T) {
		t.Parallel()
		_ = helpers.NewE2ETestContext(t, dbs.GetDriver())
		// List all available tools
		listToolsRequest := mcp.ListToolsRequest{}
		mcpListToolsResponse, err := mcpClient.ListTools(ctx, listToolsRequest)
		require.NoError(t, err, "failed to list tools")
		require.NotEmpty(t, mcpListToolsResponse.Tools, "expected at least one tool")

		// Serialize response to JSON
		responseJSON, err := json.MarshalIndent(mcpListToolsResponse, "", "  ")
		require.NoError(t, err, "failed to marshal listToolsResponse")
		t.Logf("ListTools Response:\n%s", string(responseJSON))

		// Unmarshal into our struct to check properties
		var parsed listToolsResponse
		err = json.Unmarshal(responseJSON, &parsed)
		require.NoError(t, err, "failed to unmarshal into listToolsResponse struct")

		// Assert that each tool has properties defined in inputSchema
		for _, tool := range parsed.Tools {
			require.NotNilf(t, tool.InputSchema.Properties,
				"tool %s inputSchema MUST have 'properties' field for OpenAI API compatibility",
				tool.Name)
		}
	})

}
