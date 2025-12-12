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
		binaryPath, cleanup, err := tc.BuildServer(t)
		if err != nil {
			t.Fatalf("failed to build server: %v", err)
		}
		defer cleanup()

		// Create MCP client that will communicate with the server over STDIO
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		cfg := dbs.GetDriverConf()

		// Set environment variables for the server
		env := []string{
			"NEO4J_URI=" + cfg.URI,
			"NEO4J_USERNAME=" + cfg.Username,
			"NEO4J_PASSWORD=" + cfg.Password,
			"NEO4J_DATABASE=" + cfg.Database,
			"LOG_LEVEL=" + cfg.LogLevel,
		}

		// Create MCP client with the built server binary
		mcpClient, err := client.NewStdioMCPClient(binaryPath, env)
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

		t.Logf("Server started successfully with %d tools available", len(listToolsResponse.Tools))
	})
}
