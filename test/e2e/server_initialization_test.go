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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func boolPtr(b bool) *bool {
	return &b
}

func TestServerInitializationE2E(t *testing.T) {
	ctx := context.Background()
	cfg := dbs.GetDriverConf()

	t.Run("successful initialization with all required parameters", func(t *testing.T) {
		t.Parallel()

		args := []string{
			"--uri", cfg.URI,
			"--username", cfg.Username,
			"--password", cfg.Password,
			"--database", cfg.Database,
		}

		mcpClient := helpers.NewTestClient()
		session, err := mcpClient.Connect(ctx, &mcp.CommandTransport{Command: exec.Command(server, args...)}, nil)
		require.NoError(t, err, "failed to connect MCP client")
		defer session.Close()

		// Verify server info
		initializeResult := session.InitializeResult()
		assert.Equal(t, "neo4j-mcp", initializeResult.ServerInfo.Name)
		assert.NotEmpty(t, initializeResult.ServerInfo.Version)

		// Verify capabilities
		assert.NotNil(t, initializeResult.Capabilities)
		assert.NotNil(t, initializeResult.Capabilities.Tools)

		t.Log("Server initialized successfully with expected name and capabilities")
	})

	t.Run("initialization with read-only mode enabled", func(t *testing.T) {
		t.Parallel()

		args := []string{
			"--uri", cfg.URI,
			"--username", cfg.Username,
			"--password", cfg.Password,
			"--database", cfg.Database,
			"--read-only", "true",
		}

		mcpClient := helpers.NewTestClient()
		session, err := mcpClient.Connect(ctx, &mcp.CommandTransport{Command: exec.Command(server, args...)}, nil)
		require.NoError(t, err, "failed to connect MCP client")
		defer session.Close()
		
		initializeResult := session.InitializeResult()
		assert.Equal(t, "neo4j-mcp", initializeResult.ServerInfo.Name)

		// List tools to verify read-only mode behavior
		listToolsResponse, err := session.ListTools(ctx, &mcp.ListToolsParams{})
		require.NoError(t, err, "failed to list tools in read-only mode")

		for _, tool := range listToolsResponse.Tools {
			if tool.Name == "write-cypher" {
				t.Fatal("write-cypher tool found using readOnly mode")
			}
		}
		assert.Len(t, listToolsResponse.Tools, 3, "read-only mode true returns the wrong number of tools")
	})

	t.Run("initialization with read-only mode disabled", func(t *testing.T) {
		t.Parallel()

		args := []string{
			"--uri", cfg.URI,
			"--username", cfg.Username,
			"--password", cfg.Password,
			"--database", cfg.Database,
			"--read-only", "false",
		}

		mcpClient := helpers.NewTestClient()
		session, err := mcpClient.Connect(ctx, &mcp.CommandTransport{Command: exec.Command(server, args...)}, nil)
		require.NoError(t, err, "failed to connect MCP client")
		defer session.Close()

		initializeResult := session.InitializeResult()
		assert.Equal(t, "neo4j-mcp", initializeResult.ServerInfo.Name)

		listToolsResponse, err := session.ListTools(ctx, &mcp.ListToolsParams{})
		require.NoError(t, err, "failed to list tools with read-only mode as false")
		assert.Len(t, listToolsResponse.Tools, 4, "read-only mode false returns the wrong number of tools")
	})
	t.Run("initialization with telemetry disabled", func(t *testing.T) {
		t.Parallel()

		args := []string{
			"--uri", cfg.URI,
			"--username", cfg.Username,
			"--password", cfg.Password,
			"--database", cfg.Database,
			"--telemetry", "false",
		}

		mcpClient := helpers.NewTestClient()
		session, err := mcpClient.Connect(ctx, &mcp.CommandTransport{Command: exec.Command(server, args...)}, nil)
		require.NoError(t, err, "failed to connect MCP client")
		defer session.Close()

		initializeResult := session.InitializeResult()
		assert.Equal(t, "neo4j-mcp", initializeResult.ServerInfo.Name)

		t.Log("Server initialized successfully with telemetry disabled")
	})

	t.Run("initialization with schema sample size override", func(t *testing.T) {
		t.Parallel()

		args := []string{
			"--uri", cfg.URI,
			"--username", cfg.Username,
			"--password", cfg.Password,
			"--database", cfg.Database,
			"--schema-sample-size", "50",
		}

		mcpClient := helpers.NewTestClient()
		session, err := mcpClient.Connect(ctx, &mcp.CommandTransport{Command: exec.Command(server, args...)}, nil)
		require.NoError(t, err, "failed to connect MCP client")
		defer session.Close()

		initializeResult := session.InitializeResult()
		assert.Equal(t, "neo4j-mcp", initializeResult.ServerInfo.Name)

		t.Log("Server initialized successfully with custom schema sample size")
	})

	t.Run("client initialization with invalid schema sample size", func(t *testing.T) {
		t.Parallel()

		args := []string{
			"--uri", cfg.URI,
			"--username", cfg.Username,
			"--password", cfg.Password,
			"--database", cfg.Database,
			"--schema-sample-size", "not-a-number",
		}

		mcpClient := helpers.NewTestClient()
		// Server should handle invalid schema sample size gracefully (falling back to default)
		session, err := mcpClient.Connect(ctx, &mcp.CommandTransport{Command: exec.Command(server, args...)}, nil)
		require.NoError(t, err, "failed to connect MCP client with invalid schema sample size")
		defer session.Close()

		initializeResult := session.InitializeResult()
		assert.Equal(t, "neo4j-mcp", initializeResult.ServerInfo.Name)

		t.Log("Server initialized successfully with invalid schema sample size (using default value)")
	})

	t.Run("list tools response matches tool spec definitions", func(t *testing.T) {
		t.Parallel()

		args := []string{
			"--uri", cfg.URI,
			"--username", cfg.Username,
			"--password", cfg.Password,
			"--database", cfg.Database,
		}

		mcpClient := helpers.NewTestClient()
		session, err := mcpClient.Connect(ctx, &mcp.CommandTransport{Command: exec.Command(server, args...)}, nil)
		require.NoError(t, err, "failed to connect MCP client")
		defer session.Close()

		listResponse, err := session.ListTools(ctx, &mcp.ListToolsParams{})
		require.NoError(t, err, "failed to list tools")
		require.NotEmpty(t, listResponse.Tools, "expected at least one tool")

		type propertyExpectation struct {
			jsonSchemaType string
			required       bool
		}

		type toolExpectation struct {
			description string
			annotations mcp.ToolAnnotations
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
				annotations: mcp.ToolAnnotations{
					Title:           "Read Cypher",
					ReadOnlyHint:    true,
					DestructiveHint: boolPtr(false),
					IdempotentHint:  true,
					OpenWorldHint:   boolPtr(true),
				},
				properties: map[string]propertyExpectation{
					"query":  {jsonSchemaType: "string", required: true},
					"params": {jsonSchemaType: "object", required: false},
				},
			},
			"write-cypher": {
				description: "write-cypher executes any arbitrary Cypher query, with write access, against the user-configured Neo4j database.",
				annotations: mcp.ToolAnnotations{
					Title:           "Write Cypher",
					ReadOnlyHint:    false,
					DestructiveHint: boolPtr(true),
					IdempotentHint:  false,
					OpenWorldHint:   boolPtr(true),
				},
				properties: map[string]propertyExpectation{
					"query":  {jsonSchemaType: "string", required: true},
					"params": {jsonSchemaType: "object", required: false},
				},
			},
			"get-schema": {
				annotations: mcp.ToolAnnotations{
					Title:           "Get Neo4j Schema",
					ReadOnlyHint:    true,
					DestructiveHint: boolPtr(false),
					IdempotentHint:  true,
					OpenWorldHint:   boolPtr(true),
				},
				properties: map[string]propertyExpectation{},
			},
			"list-gds-procedures": {
				annotations: mcp.ToolAnnotations{
					Title:           "List available Neo4j GDS procedures",
					ReadOnlyHint:    true,
					DestructiveHint: boolPtr(false),
					IdempotentHint:  true,
					OpenWorldHint:   boolPtr(true),
				},
				properties: map[string]propertyExpectation{},
			},
		}

		advertised := make(map[string]*mcp.Tool, len(listResponse.Tools))
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

				require.NotNilf(t, tool.Annotations.DestructiveHint, "tool %q is missing destructiveHint annotation", name)
				require.NotNilf(t, tool.Annotations.OpenWorldHint, "tool %q is missing openWorldHint annotation", name)

				assert.Equalf(t, exp.annotations.Title, tool.Annotations.Title,
					"annotations.title mismatch for %q", name)
				assert.Equalf(t, exp.annotations.ReadOnlyHint, tool.Annotations.ReadOnlyHint,
					"annotations.readOnlyHint mismatch for %q", name)
				assert.Equalf(t, *exp.annotations.DestructiveHint, *tool.Annotations.DestructiveHint,
					"annotations.destructiveHint mismatch for %q", name)
				assert.Equalf(t, exp.annotations.IdempotentHint, tool.Annotations.IdempotentHint,
					"annotations.idempotentHint mismatch for %q", name)
				assert.Equalf(t, *exp.annotations.OpenWorldHint, *tool.Annotations.OpenWorldHint,
					"annotations.openWorldHint mismatch for %q", name)

				schema, ok := tool.InputSchema.(map[string]any)
				require.Truef(t, ok, "inputSchema for %q should unmarshal to a map, got %T", name, tool.InputSchema)

				assert.Equalf(t, "object", schema["type"],
					"inputSchema.type for %q must be \"object\" per MCP spec", name)

				properties, _ := schema["properties"].(map[string]any)
				require.NotNilf(t, properties,
					"inputSchema.properties for %q must be present per MCP spec", name)

				// Every expected property must appear in inputSchema.properties
				// with the declared JSON Schema type.
				var expectedRequired []string
				for propName, propExp := range exp.properties {
					raw, ok := properties[propName]
					if !assert.Truef(t, ok,
						"tool %q is missing expected input property %q (properties=%v)",
						name, propName, properties) {
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
				for advertisedProp := range properties {
					_, known := exp.properties[advertisedProp]
					assert.Truef(t, known,
						"tool %q advertises unexpected input property %q",
						name, advertisedProp)
				}

				var actualRequired []string
				if requiredRaw, ok := schema["required"].([]any); ok {
					for _, required := range requiredRaw {
						if requiredName, ok := required.(string); ok {
							actualRequired = append(actualRequired, requiredName)
						}
					}
				}

				assert.ElementsMatchf(t, expectedRequired, actualRequired,
					"inputSchema.required for %q does not match spec-declared required fields",
					name)

			})
		}

	})
}
