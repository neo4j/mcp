// Copyright (c) "Neo4j"
// Neo4j Sweden AB [http://neo4j.com]

//go:build e2e

package e2e

import (
	"context"
	"testing"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/neo4j/mcp/test/e2e/helpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServerInitializationE2E(t *testing.T) {
	ctx := context.Background()
	cfg := dbs.GetDriverConf()

	t.Run("successful initialization with all required parameters", func(t *testing.T) {
		t.Parallel()

		args := []string{
			"--neo4j-uri", cfg.URI,
			"--neo4j-username", cfg.Username,
			"--neo4j-password", cfg.Password,
			"--neo4j-database", cfg.Database,
		}

		mcpClient, err := client.NewStdioMCPClient(server, []string{}, args...)
		require.NoError(t, err, "failed to create MCP client")

		defer mcpClient.Close()

		// Test initialization
		initRequest := helpers.BuildInitializeRequest()
		initResponse, err := mcpClient.Initialize(ctx, initRequest)
		require.NoError(t, err, "failed to initialize MCP server")

		// Verify server info
		assert.Equal(t, "neo4j-mcp", initResponse.ServerInfo.Name)
		assert.NotEmpty(t, initResponse.ServerInfo.Version)

		// Verify capabilities
		assert.NotNil(t, initResponse.Capabilities)
		assert.NotNil(t, initResponse.Capabilities.Tools)

		t.Log("Server initialized successfully with expected name and capabilities")
	})

	t.Run("initialization without a database name", func(t *testing.T) {
		t.Parallel()

		args := []string{
			"--neo4j-uri", cfg.URI,
			"--neo4j-username", cfg.Username,
			"--neo4j-password", cfg.Password,
		}

		mcpClient, err := client.NewStdioMCPClient(server, []string{}, args...)
		require.NoError(t, err, "failed to create MCP client")

		defer mcpClient.Close()

		// Test should pass as the default database is neo4j
		initRequest := helpers.BuildInitializeRequest()
		initResponse, _ := mcpClient.Initialize(ctx, initRequest)
		assert.Equal(t, "neo4j-mcp", initResponse.ServerInfo.Name)

	})

	t.Run("initialization with read-only mode enabled", func(t *testing.T) {
		t.Parallel()

		args := []string{
			"--neo4j-uri", cfg.URI,
			"--neo4j-username", cfg.Username,
			"--neo4j-password", cfg.Password,
			"--neo4j-database", cfg.Database,
			"--neo4j-read-only", "true",
		}

		mcpClient, err := client.NewStdioMCPClient(server, []string{}, args...)
		require.NoError(t, err, "failed to create MCP client")

		defer mcpClient.Close()

		// Test initialization in read-only mode
		initRequest := helpers.BuildInitializeRequest()
		initResponse, err := mcpClient.Initialize(ctx, initRequest)
		require.NoError(t, err, "failed to initialize MCP server in read-only mode")

		assert.Equal(t, "neo4j-mcp", initResponse.ServerInfo.Name)

		// List tools to verify read-only mode behavior
		listToolsResponse, err := mcpClient.ListTools(ctx, mcp.ListToolsRequest{})
		require.NoError(t, err, "failed to list tools in read-only mode")

		for _, tool := range listToolsResponse.Tools {
			if tool.Name == "write-cypher" {
				t.Fatal("write-cypher tool found using readonly mode")
			}
		}
		assert.Len(t, listToolsResponse.Tools, 3, "read-only mode true returns the wrong number of tools")
	})

	t.Run("initialization with read-only mode disabled", func(t *testing.T) {
		t.Parallel()

		args := []string{
			"--neo4j-uri", cfg.URI,
			"--neo4j-username", cfg.Username,
			"--neo4j-password", cfg.Password,
			"--neo4j-database", cfg.Database,
			"--neo4j-read-only", "false",
		}

		mcpClient, err := client.NewStdioMCPClient(server, []string{}, args...)
		require.NoError(t, err, "failed to create MCP client")

		defer mcpClient.Close()

		initRequest := helpers.BuildInitializeRequest()
		initResponse, err := mcpClient.Initialize(ctx, initRequest)
		require.NoError(t, err, "failed to initialize MCP server in read-only mode")

		assert.Equal(t, "neo4j-mcp", initResponse.ServerInfo.Name)

		listToolsResponse, err := mcpClient.ListTools(ctx, mcp.ListToolsRequest{})
		require.NoError(t, err, "failed to list tools with read-only mode as false")
		assert.Len(t, listToolsResponse.Tools, 4, "read-only mode false returns the wrong number of tools")
	})
	t.Run("initialization with telemetry disabled", func(t *testing.T) {
		t.Parallel()

		args := []string{
			"--neo4j-uri", cfg.URI,
			"--neo4j-username", cfg.Username,
			"--neo4j-password", cfg.Password,
			"--neo4j-database", cfg.Database,
			"--neo4j-telemetry", "false",
		}

		mcpClient, err := client.NewStdioMCPClient(server, []string{}, args...)
		require.NoError(t, err, "failed to create MCP client")

		defer mcpClient.Close()

		// Test initialization with telemetry disabled
		initRequest := helpers.BuildInitializeRequest()
		initResponse, err := mcpClient.Initialize(ctx, initRequest)
		require.NoError(t, err, "failed to initialize MCP server with telemetry disabled")

		assert.Equal(t, "neo4j-mcp", initResponse.ServerInfo.Name)

		t.Log("Server initialized successfully with telemetry disabled")
	})

	t.Run("initialization with schema sample size override", func(t *testing.T) {
		t.Parallel()

		args := []string{
			"--neo4j-uri", cfg.URI,
			"--neo4j-username", cfg.Username,
			"--neo4j-password", cfg.Password,
			"--neo4j-database", cfg.Database,
			"--neo4j-schema-sample-size", "50",
		}

		mcpClient, err := client.NewStdioMCPClient(server, []string{}, args...)
		require.NoError(t, err, "failed to create MCP client")

		defer mcpClient.Close()

		// Test initialization with custom schema sample size
		initRequest := helpers.BuildInitializeRequest()
		initResponse, err := mcpClient.Initialize(ctx, initRequest)
		require.NoError(t, err, "failed to initialize MCP server with custom schema sample size")

		assert.Equal(t, "neo4j-mcp", initResponse.ServerInfo.Name)

		t.Log("Server initialized successfully with custom schema sample size")
	})

	t.Run("client initialization with invalid schema sample size", func(t *testing.T) {
		t.Parallel()

		args := []string{
			"--neo4j-uri", cfg.URI,
			"--neo4j-username", cfg.Username,
			"--neo4j-password", cfg.Password,
			"--neo4j-database", cfg.Database,
			"--neo4j-schema-sample-size", "not-a-number",
		}

		mcpClient, err := client.NewStdioMCPClient(server, []string{}, args...)
		require.NoError(t, err, "failed to create MCP client")

		defer mcpClient.Close()

		// Server should handle invalid schema sample size gracefully (falling back to default)
		initRequest := helpers.BuildInitializeRequest()
		initResponse, err := mcpClient.Initialize(ctx, initRequest)
		require.NoError(t, err, "failed to initialize MCP server with invalid schema sample size")

		assert.Equal(t, "neo4j-mcp", initResponse.ServerInfo.Name)

		t.Log("Server initialized successfully with invalid schema sample size (using default value)")
	})

	t.Run("list tools response matches tool spec definitions", func(t *testing.T) {
		t.Parallel()

		args := []string{
			"--neo4j-uri", cfg.URI,
			"--neo4j-username", cfg.Username,
			"--neo4j-password", cfg.Password,
			"--neo4j-database", cfg.Database,
		}

		mcpClient, err := client.NewStdioMCPClient(server, []string{}, args...)
		require.NoError(t, err, "failed to create MCP client")
		defer mcpClient.Close()

		_, err = mcpClient.Initialize(ctx, helpers.BuildInitializeRequest())
		require.NoError(t, err, "failed to initialize MCP server")

		listResponse, err := mcpClient.ListTools(ctx, mcp.ListToolsRequest{})
		require.NoError(t, err, "failed to list tools")
		require.NotEmpty(t, listResponse.Tools, "expected at least one tool")

		type propertyExpectation struct {
			jsonSchemaType string
			required       bool
		}

		type toolExpectation struct {
			description string
			annotations mcp.ToolAnnotation
			// properties is the set of property names that MUST be present
			// on inputSchema.properties, keyed by property name. The value
			// describes additional per-property expectations. (see issue #157 for clear indication on why)
			properties map[string]propertyExpectation
		}

		// The expectations below are derived from the *_spec.go files under internal/tools.
		// They represent what each tool is intended to advertise according to the latest MCP spec
		// (tools/list response shape: name, description, inputSchema with type/properties/required, and tool annotations).
		expected := map[string]toolExpectation{
			"read-cypher": {
				description: "read-cypher can run only read-only Cypher statements. For write operations (CREATE, MERGE, DELETE, SET, etc...), schema/admin commands, or PROFILE queries, use write-cypher instead.",
				annotations: mcp.ToolAnnotation{
					Title:           "Read Cypher",
					ReadOnlyHint:    mcp.ToBoolPtr(true),
					DestructiveHint: mcp.ToBoolPtr(false),
					IdempotentHint:  mcp.ToBoolPtr(true),
					OpenWorldHint:   mcp.ToBoolPtr(true),
				},
				properties: map[string]propertyExpectation{
					"query":  {jsonSchemaType: "string", required: true},
					"params": {jsonSchemaType: "object", required: false},
				},
			},
			"write-cypher": {
				description: "write-cypher executes any arbitrary Cypher query, with write access, against the user-configured Neo4j database.",
				annotations: mcp.ToolAnnotation{
					Title:           "Write Cypher",
					ReadOnlyHint:    mcp.ToBoolPtr(false),
					DestructiveHint: mcp.ToBoolPtr(true),
					IdempotentHint:  mcp.ToBoolPtr(false),
					OpenWorldHint:   mcp.ToBoolPtr(true),
				},
				properties: map[string]propertyExpectation{
					"query":  {jsonSchemaType: "string", required: true},
					"params": {jsonSchemaType: "object", required: false},
				},
			},
			"get-schema": {
				annotations: mcp.ToolAnnotation{
					Title:           "Get Neo4j Schema",
					ReadOnlyHint:    mcp.ToBoolPtr(true),
					DestructiveHint: mcp.ToBoolPtr(false),
					IdempotentHint:  mcp.ToBoolPtr(true),
					OpenWorldHint:   mcp.ToBoolPtr(true),
				},
				properties: map[string]propertyExpectation{
					"properties": {jsonSchemaType: "object", required: true},
				},
			},
			"list-gds-procedures": {
				annotations: mcp.ToolAnnotation{
					Title:           "List available Neo4j GDS procedures",
					ReadOnlyHint:    mcp.ToBoolPtr(true),
					DestructiveHint: mcp.ToBoolPtr(false),
					IdempotentHint:  mcp.ToBoolPtr(true),
					OpenWorldHint:   mcp.ToBoolPtr(true),
				},
				properties: map[string]propertyExpectation{
					"properties": {jsonSchemaType: "object", required: true},
				},
			},
		}

		advertised := make(map[string]mcp.Tool, len(listResponse.Tools))
		for _, tool := range listResponse.Tools {
			advertised[tool.Name] = tool
		}

		for name, exp := range expected {
			t.Run(name, func(t *testing.T) {
				tool, ok := advertised[name]
				require.Truef(t, ok, "expected tool %q to be advertised by the server", name)

				if exp.description != "" {
					assert.Equalf(t, exp.description, tool.Description,
						"description for %q does not match spec", name)
				}

				require.NotNilf(t, tool.Annotations.ReadOnlyHint, "tool %q is missing readOnlyHint annotation", name)
				require.NotNilf(t, tool.Annotations.DestructiveHint, "tool %q is missing destructiveHint annotation", name)
				require.NotNilf(t, tool.Annotations.IdempotentHint, "tool %q is missing idempotentHint annotation", name)
				require.NotNilf(t, tool.Annotations.OpenWorldHint, "tool %q is missing openWorldHint annotation", name)

				assert.Equalf(t, exp.annotations.Title, tool.Annotations.Title,
					"annotations.title mismatch for %q", name)
				assert.Equalf(t, *exp.annotations.ReadOnlyHint, *tool.Annotations.ReadOnlyHint,
					"annotations.readOnlyHint mismatch for %q", name)
				assert.Equalf(t, *exp.annotations.DestructiveHint, *tool.Annotations.DestructiveHint,
					"annotations.destructiveHint mismatch for %q", name)
				assert.Equalf(t, *exp.annotations.IdempotentHint, *tool.Annotations.IdempotentHint,
					"annotations.idempotentHint mismatch for %q", name)
				assert.Equalf(t, *exp.annotations.OpenWorldHint, *tool.Annotations.OpenWorldHint,
					"annotations.openWorldHint mismatch for %q", name)

				assert.Equalf(t, "object", tool.InputSchema.Type,
					"inputSchema.type for %q must be \"object\" per MCP spec", name)
				require.NotNilf(t, tool.InputSchema.Properties,
					"inputSchema.properties for %q must be present per MCP spec", name)

				// Every expected property must appear in inputSchema.properties
				// with the declared JSON Schema type.
				var expectedRequired []string
				for propName, propExp := range exp.properties {
					raw, ok := tool.InputSchema.Properties[propName]
					if !assert.Truef(t, ok,
						"tool %q is missing expected input property %q (properties=%v)",
						name, propName, tool.InputSchema.Properties) {
						continue
					}

					if propExp.jsonSchemaType != "" {
						propMap, ok := raw.(map[string]any)
						if assert.Truef(t, ok,
							"property %q on %q should be a JSON Schema object, got %T",
							propName, name, raw) {
							assert.Equalf(t, propExp.jsonSchemaType, propMap["type"],
								"property %q on %q should have JSON Schema type %q",
								propName, name, propExp.jsonSchemaType)
						}
					}

					if propExp.required {
						expectedRequired = append(expectedRequired, propName)
					}
				}

				// inputSchema.properties should not advertise fields that
				// aren't declared in the spec's input struct.
				for advertisedProp := range tool.InputSchema.Properties {
					_, known := exp.properties[advertisedProp]
					assert.Truef(t, known,
						"tool %q advertises unexpected input property %q",
						name, advertisedProp)
				}

				assert.ElementsMatchf(t, expectedRequired, tool.InputSchema.Required,
					"inputSchema.required for %q does not match spec-declared required fields",
					name)

			})
		}

	})
}
