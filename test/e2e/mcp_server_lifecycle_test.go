//go:build e2e

package e2e

import (
	"context"
	"testing"
	"time"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/neo4j/mcp/test/e2e/helpers"
)

func TestSeverMCP(t *testing.T) {
	t.Parallel()
	t.Run("lifecycle test (MCPServer -> MCP Client -> Initialize Req -> List Tools -> Call Tool -> Stop)", func(t *testing.T) {
		tc := helpers.NewE2ETestContext(t, dbs.GetDriver())

		// Build the server binary
		// binaryPath, cleanup, err := tc.BuildServer(t)
		// if err != nil {
		// 	t.Fatalf("failed to build server: %v", err)
		// }
		// defer cleanup()

		// Create MCP client that will communicate with the server over STDIO
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		cfg := dbs.GetDriverConf()

		// Prepare CLI arguments for the server (thread-safe approach)
		args := []string{
			"--neo4j-uri", cfg.URI,
			"--neo4j-username", cfg.Username,
			"--neo4j-password", cfg.Password,
			"--neo4j-database", cfg.Database,
		}

		// Create MCP client with the built server binary and CLI arguments
		mcpClient, err := client.NewStdioMCPClient(server, []string{}, args...)
		if err != nil {
			t.Fatalf("failed to create MCP client: %v", err)
		}
		defer mcpClient.Close()

		// Test server initialization
		initializeResponse, err := mcpClient.Initialize(ctx, tc.BuildInitializeRequest())
		if err != nil {
			t.Fatalf("failed to initialize MCP server: %v", err)
		}

		expectedServerInfoName := "neo4j-mcp"
		if initializeResponse.ServerInfo.Name != expectedServerInfoName {
			t.Fatalf("expected server name returned from initialize request to be: %s, but found: %s", expectedServerInfoName, initializeResponse.ServerInfo.Name)
		}

		// Test basic functionality - list tools
		listToolsResponse, err := mcpClient.ListTools(ctx, mcp.ListToolsRequest{})
		if err != nil {
			t.Fatalf("failed to list tools: %v", err)
		}

		// Verify we have the expected tools
		if len(listToolsResponse.Tools) == 0 {
			t.Fatal("expected tools to be available, but got none")
		}

		// Test calling a tool, get-schema for simplicity.
		callToolRequest := mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Name: "get-schema",
			},
		}

		callToolResponse, err := mcpClient.CallTool(ctx, callToolRequest)
		if err != nil {
			t.Fatalf("failed to call get-schema tool: %v", err)
		}

		// Verify the tool call was successful
		if callToolResponse.IsError {
			textContent, ok := mcp.AsTextContent(callToolResponse.Content[0])
			if !ok {
				t.Fatalf("expected error as TextContent, got %T", callToolResponse.Content[0])
			}
			t.Fatalf("get-schema tool call returned an error: %s", textContent.Text)
		}

		if len(callToolResponse.Content) == 0 {
			t.Fatal("expected get-schema tool to return content, but got none")
		}

		t.Logf("Server started successfully with %d tools available", len(listToolsResponse.Tools))
		t.Logf("Successfully called get-schema tool and received %d content items", len(callToolResponse.Content))
	})
}
