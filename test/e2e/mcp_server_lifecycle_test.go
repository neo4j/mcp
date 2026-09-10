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
)

func TestSeverLifecycleMCPE2E(t *testing.T) {
	t.Parallel()

	t.Run("lifecycle test (MCPServer -> MCP Client -> Initialize Req -> List Tools -> Call Tool -> Stop)", func(t *testing.T) {
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
		helpers.NewE2ETestContext(t, dbs.GetDriver())

		// Test server initialization
		initializeResult := session.InitializeResult()

		expectedServerInfoName := "neo4j-mcp"
		if initializeResult.ServerInfo.Name != expectedServerInfoName {
			t.Fatalf("expected server name returned from initialize request to be: %s, but found: %s", expectedServerInfoName, initializeResult.ServerInfo.Name)
		}

		// Test basic functionality - list tools
		listToolsResponse, err := session.ListTools(ctx, &mcp.ListToolsParams{})
		if err != nil {
			t.Fatalf("failed to list tools: %v", err)
		}

		// Verify we have the expected tools
		if len(listToolsResponse.Tools) == 0 {
			t.Fatal("expected tools to be available, but got none")
		}

		// Test calling a tool, get-schema for simplicity.
		callToolResponse, err := session.CallTool(ctx, &mcp.CallToolParams{
			Name: "get-schema",
		})
		if err != nil {
			t.Fatalf("failed to call get-schema tool: %v", err)
		}

		// Verify the tool call was successful
		if callToolResponse.IsError {
			textContent, ok := callToolResponse.Content[0].(*mcp.TextContent)
			if !ok {
				t.Fatalf("expected error as TextContent, got %T", callToolResponse.Content[0])
			}
			t.Fatalf("get-schema tool call returned an error: %s", textContent.Text)
		}

		if len(callToolResponse.Content) == 0 {
			t.Fatal("expected get-schema tool to return content, but got none")
		}
		defer session.Close()
		t.Logf("Server started successfully with %d tools available", len(listToolsResponse.Tools))
		t.Logf("Successfully called get-schema tool and received %d content items", len(callToolResponse.Content))

	})
}
