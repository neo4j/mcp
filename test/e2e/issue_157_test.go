// Copyright (c) "Neo4j"
// Neo4j Sweden AB [http://neo4j.com]

//go:build e2e

package e2e

import (
	"context"
	"encoding/json"
	"os/exec"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
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
		"--uri", cfg.URI,
		"--username", cfg.Username,
		"--password", cfg.Password,
		"--database", cfg.Database,
	}

	mcpClient := helpers.NewTestClient()
	session, err := mcpClient.Connect(ctx, &mcp.CommandTransport{Command: exec.Command(server, args...)}, nil)
	if err != nil {
		t.Fatalf("failed to connect MCP client: %v", err)
	}
	t.Cleanup(func() {
		session.Close()
	})

	t.Run("all tools returned from listTools should contains inputSchema properties", func(t *testing.T) {
		t.Parallel()
		_ = helpers.NewE2ETestContext(t, dbs.GetDriver())
		// List all available tools
		mcpListToolsResponse, err := session.ListTools(ctx, &mcp.ListToolsParams{})
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
