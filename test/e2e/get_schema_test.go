//go:build e2e

package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/neo4j/mcp/test/e2e/helpers"
)

func TestGetSchemaE2E(t *testing.T) {
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
	t.Run("get-schema with empty database", func(t *testing.T) {
		t.Parallel()
		// Call get-schema tool on empty database
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

		// For empty database, should return a message indicating no schema
		if len(callToolResponse.Content) == 0 {
			t.Fatal("expected get-schema tool to return content, but got none")
		}

		textContent, ok := mcp.AsTextContent(callToolResponse.Content[0])
		if !ok {
			t.Fatalf("expected content as TextContent, got %T", callToolResponse.Content[0])
		}

		// Should contain message about empty database
		expectedMessage := "The get-schema tool executed successfully; however, since the Neo4j instance contains no data, no schema information was returned."
		if textContent.Text != expectedMessage {
			t.Fatalf("expected empty database message, got: %s", textContent.Text)
		}

		t.Log("Successfully handled get-schema on empty database")
	})

	t.Run("get-schema with nodes only", func(t *testing.T) {
		t.Parallel()
		tc := helpers.NewE2ETestContext(t, dbs.GetDriver())

		// Seed test data - create nodes with different properties
		personLabel, err := tc.SeedNode("Person", map[string]any{
			"name": "Alice",
			"age":  30,
		})
		if err != nil {
			t.Fatalf("failed to seed Person node: %v", err)
		}

		companyLabel, err := tc.SeedNode("Company", map[string]any{
			"name":    "Neo4j",
			"founded": 2007,
			"active":  true,
		})
		if err != nil {
			t.Fatalf("failed to seed Company node: %v", err)
		}

		// Call get-schema tool
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

		textContent, ok := mcp.AsTextContent(callToolResponse.Content[0])
		if !ok {
			t.Fatalf("expected content as TextContent, got %T", callToolResponse.Content[0])
		}

		// Verify the JSON response contains expected schema entries
		schemaJSON := textContent.Text

		// Check that the response is valid JSON and contains expected entries
		if !json.Valid([]byte(schemaJSON)) {
			t.Fatalf("get-schema returned invalid JSON: %s", schemaJSON)
		}

		// Verify that our seeded labels appear in the schema
		if !strings.Contains(schemaJSON, personLabel.String()) {
			t.Errorf("schema JSON should contain Person label %q, got: %s", personLabel.String(), schemaJSON)
		}

		if !strings.Contains(schemaJSON, companyLabel.String()) {
			t.Errorf("schema JSON should contain Company label %q, got: %s", companyLabel.String(), schemaJSON)
		}

		// Verify that the schema contains expected property types
		expectedPatterns := []string{
			`"type":"node"`,       // Both nodes should be of type "node"
			`"name":"STRING"`,     // Both have name properties of type STRING
			`"age":"INTEGER"`,     // Person has age of type INTEGER
			`"founded":"INTEGER"`, // Company has founded of type INTEGER
			`"active":"BOOLEAN"`,  // Company has active of type BOOLEAN
		}

		for _, pattern := range expectedPatterns {
			if !strings.Contains(schemaJSON, pattern) {
				t.Errorf("schema JSON should contain pattern %q, got: %s", pattern, schemaJSON)
			}
		}

		t.Logf("Successfully retrieved schema JSON: %s", schemaJSON)
	})

	t.Run("get-schema with nodes and relationships", func(t *testing.T) {
		t.Parallel()
		tc := helpers.NewE2ETestContext(t, dbs.GetDriver())

		// Seed test data - create nodes and relationships
		personLabel, err := tc.SeedNode("Person", map[string]any{
			"name": "Bob",
			"age":  25,
		})
		if err != nil {
			t.Fatalf("failed to seed Person node: %v", err)
		}

		companyLabel, err := tc.SeedNode("Company", map[string]any{
			"name": "TechCorp",
		})
		if err != nil {
			t.Fatalf("failed to seed Company node: %v", err)
		}

		// Create a relationship between person and company
		relationshipLabel := tc.GetUniqueLabel("WORKS_FOR")
		relationshipQuery := fmt.Sprintf(
			"MATCH (p:%s), (c:%s) CREATE (p)-[r:%s {since: 2020, position: 'Developer'}]->(c)",
			personLabel, companyLabel, relationshipLabel,
		)
		_, err = tc.Service.ExecuteWriteQuery(context.Background(), relationshipQuery, map[string]any{})
		if err != nil {
			t.Fatalf("failed to create relationship: %v", err)
		}

		// Call get-schema tool
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

		textContent, ok := mcp.AsTextContent(callToolResponse.Content[0])
		if !ok {
			t.Fatalf("expected content as TextContent, got %T", callToolResponse.Content[0])
		}

		// Verify the JSON response contains expected schema entries
		schemaJSON := textContent.Text

		// Check that the response is valid JSON
		if !json.Valid([]byte(schemaJSON)) {
			t.Fatalf("get-schema returned invalid JSON: %s", schemaJSON)
		}

		// Verify that our seeded labels appear in the schema
		if !strings.Contains(schemaJSON, personLabel.String()) {
			t.Errorf("schema JSON should contain Person label %q, got: %s", personLabel.String(), schemaJSON)
		}

		if !strings.Contains(schemaJSON, companyLabel.String()) {
			t.Errorf("schema JSON should contain Company label %q, got: %s", companyLabel.String(), schemaJSON)
		}

		if !strings.Contains(schemaJSON, relationshipLabel.String()) {
			t.Errorf("schema JSON should contain relationship label %q, got: %s", relationshipLabel.String(), schemaJSON)
		}

		// Verify that the schema contains expected patterns for nodes and relationships
		expectedPatterns := []string{
			`"type":"node"`,         // Nodes should be of type "node"
			`"type":"relationship"`, // Relationship should be of type "relationship"
			`"name":"STRING"`,       // Name properties
			`"age":"INTEGER"`,       // Person age property
			`"since":"INTEGER"`,     // Relationship since property
			`"position":"STRING"`,   // Relationship position property
			`"direction":"out"`,     // Relationship direction
		}

		for _, pattern := range expectedPatterns {
			if !strings.Contains(schemaJSON, pattern) {
				t.Errorf("schema JSON should contain pattern %q, got: %s", pattern, schemaJSON)
			}
		}

		t.Logf("Successfully retrieved schema with nodes and relationships: %s", schemaJSON)
	})

	t.Run("get-schema tool availability", func(t *testing.T) {
		t.Parallel()
		helpers.NewE2ETestContext(t, dbs.GetDriver())

		// List tools to verify get-schema is available
		listToolsResponse, err := mcpClient.ListTools(ctx, mcp.ListToolsRequest{})
		if err != nil {
			t.Fatalf("failed to list tools: %v", err)
		}

		// Verify get-schema tool is in the list
		getSchemaToolFound := false
		for _, tool := range listToolsResponse.Tools {
			if tool.Name == "get-schema" {
				getSchemaToolFound = true

				// Verify tool description is not empty
				if tool.Description == "" {
					t.Error("get-schema tool should have a description")
				}

				t.Logf("Found get-schema tool with description: %s", tool.Description)
				break
			}
		}

		if !getSchemaToolFound {
			t.Fatal("get-schema tool not found in available tools")
		}

		t.Logf("Successfully verified get-schema tool availability among %d total tools", len(listToolsResponse.Tools))
	})
}
