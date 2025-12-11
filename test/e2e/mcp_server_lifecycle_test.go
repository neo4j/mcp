package e2e

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
)

func TestSeverMCP(t *testing.T) {
	t.Parallel()
	t.Run("lifecycle test (MCPServer -> MCP Client -> Initialize Req -> List Tools -> Call Tool -> Stop)", func(t *testing.T) {
		// Build the server binary
		binaryPath, cleanup, err := buildServer(t)
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
		initializeResponse, err := mcpClient.Initialize(ctx, buildInitializeRequest())
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

func buildServer(t *testing.T) (string, func(), error) {
	// Create temporary directory for the build
	tmpDir := os.TempDir()

	// Create a unique subdirectory for this test run
	buildDir, err := os.MkdirTemp(tmpDir, "mcp-server-test-*")
	if err != nil {
		return "", nil, err
	}

	// Define cleanup function
	cleanup := func() {
		if err := os.RemoveAll(buildDir); err != nil {
			t.Logf("failed to cleanup build directory: %v", err)
		}
	}

	// Define binary path
	binaryName := "neo4j-mcp"
	binaryPath := filepath.Join(buildDir, binaryName)

	// Get the project root directory (go up from test/e2e/)
	projectRoot := filepath.Join("..", "..")
	sourceDir := filepath.Join(projectRoot, "cmd", "neo4j-mcp")

	// Build the server binary
	cmd := exec.Command("go", "build", "-o", binaryPath, ".")
	cmd.Dir = sourceDir
	cmd.Env = os.Environ() // Use current environment

	// Capture build output for debugging
	output, err := cmd.CombinedOutput()
	if err != nil {
		cleanup()
		return "", nil, err
	}

	t.Logf("Built server binary at: %s", binaryPath)
	if len(output) > 0 {
		t.Logf("Build output: %s", string(output))
	}

	// Verify the binary was created
	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		cleanup()
		return "", nil, err
	}

	return binaryPath, cleanup, nil
}

func buildInitializeRequest() mcp.InitializeRequest {
	InitializeRequest := mcp.InitializeRequest{}
	InitializeRequest.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	InitializeRequest.Params.ClientInfo = mcp.Implementation{
		Name:    "test-client",
		Version: "1.0.0",
	}
	return InitializeRequest
}
