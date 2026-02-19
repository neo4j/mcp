//
// Copyright (c) "Neo4j"
// Neo4j Sweden AB [http://neo4j.com]

//go:build e2e

package e2e

import (
	"context"
	"fmt"
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
		textContent, ok := mcp.AsTextContent(callToolResponse.Content[0])
		if !ok {
			t.Fatalf("expected error as TextContent, got %T", callToolResponse.Content[0])
		}
		// Verify the tool call was successful
		if callToolResponse.IsError {

			t.Fatalf("get-schema tool call returned an error: %s", textContent.Text)
		}

		if len(callToolResponse.Content) == 0 {
			t.Fatal("expected get-schema tool to return content, but got none")
		}

		// Parse and validate the JSON schema response
		schemaJSON := textContent.Text

		personExpectation := map[string]interface{}{
			"key": personLabel.String(),
			"value": map[string]interface{}{
				"type": "node",
				"properties": map[string]interface{}{
					"name": "STRING",
					"age":  "INTEGER",
				},
			},
		}
		companyExpectation := map[string]interface{}{
			"key": companyLabel.String(),
			"value": map[string]interface{}{
				"type": "node",
				"properties": map[string]interface{}{
					"name":    "STRING",
					"founded": "INTEGER",
					"active":  "BOOLEAN",
				},
			},
		}
		tc.AssertJSONListContainsObject(schemaJSON, personExpectation)
		tc.AssertJSONListContainsObject(schemaJSON, companyExpectation)

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

		// Parse and validate the JSON schema response
		schemaJSON := textContent.Text

		// Create expected schema entries
		personExpectation := map[string]interface{}{
			"key": personLabel.String(),
			"value": map[string]interface{}{
				"type": "node",
				"properties": map[string]interface{}{
					"name": "STRING",
					"age":  "INTEGER",
				},
				"relationships": map[string]interface{}{
					relationshipLabel.String(): map[string]interface{}{
						"direction":  "out",
						"labels":     []interface{}{companyLabel.String()},
						"properties": map[string]interface{}{"position": "STRING", "since": "INTEGER"},
					},
				},
			},
		}

		companyExpectation := map[string]interface{}{
			"key": companyLabel.String(),
			"value": map[string]interface{}{
				"type": "node",
				"properties": map[string]interface{}{
					"name": "STRING",
				},
				"relationships": map[string]interface{}{
					relationshipLabel.String(): map[string]interface{}{
						"direction":  "in",
						"labels":     []interface{}{personLabel.String()},
						"properties": map[string]interface{}{"position": "STRING", "since": "INTEGER"},
					},
				},
			},
		}

		relationshipExpectation := map[string]interface{}{
			"key": relationshipLabel.String(),
			"value": map[string]interface{}{
				"type":       "relationship",
				"properties": map[string]interface{}{"position": "STRING", "since": "INTEGER"},
			},
		}

		// Assert all expected entries exist in the schema
		tc.AssertJSONListContainsObject(schemaJSON, personExpectation)
		tc.AssertJSONListContainsObject(schemaJSON, companyExpectation)
		tc.AssertJSONListContainsObject(schemaJSON, relationshipExpectation)

		t.Logf("Successfully retrieved schema with nodes and relationships: %s", schemaJSON)
	})
}
