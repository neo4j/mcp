# Integration Tests

Integration tests for the Neo4j MCP server using a shared Neo4j container (includes APOC + GDS).

## Quick Start

```go
func TestMCPIntegration_MyFeature(t *testing.T) {
    t.Parallel()
    tc := NewTestContext(t)

    // Seed test data (automatically isolated and cleaned up)
    tc.SeedNode("Person", map[string]any{"name": "Alice"})

    // Call tool
    handler := tools.ReadCypherHandler(tc.Deps)
    res := tc.CallTool(handler, map[string]any{
        "query":  "MATCH (p:Person {test_id: $testID}) RETURN p",
        "params": map[string]any{"testID": tc.TestID},
    })

    // Parse and assert
    var records []map[string]any
    tc.ParseJSONResponse(res, &records)

    person := records[0]["p"].(map[string]any)
    AssertNodeProperties(t, person, map[string]any{"name": "Alice"})
}
```

## Key Helpers

**TestContext:**

- `NewTestContext(t)` - Auto-isolation + cleanup
- `SeedNode(label, props)` - Create test data
- `CallTool(handler, args)` - Invoke MCP tool
- `ParseJSONResponse(res, &v)` - Parse response
- `VerifyNodeInDB(label, props)` - Check DB state

**Assertions:**

- `AssertNodeProperties(t, node, props)`
- `AssertNodeHasLabel(t, node, label)`
- `AssertSchemaHasNodeType(t, schema, label, props)`

## Running Tests

```bash
go test -tags=integration ./test/integration/... -v              # All tests
go test -tags=integration ./test/integration/... -run MyFeature  # Specific test
go test -tags=integration ./test/integration/... -race           # With race detection
```

## Configuration

The integration tests use environment variables to configure the Neo4j test container. All variables have sensible defaults:

| Environment Variable | Default                         | Description               |
| -------------------- | ------------------------------- | ------------------------- |
| `NEO4J_IMAGE`        | `neo4j:5.24.2-community`        | Neo4j Docker image to use |
| `NEO4J_USERNAME`     | `neo4j`                         | Database username         |
| `NEO4J_PASSWORD`     | `password`                      | Database password         |
| `NEO4JLABS_PLUGINS`  | `["apoc","graph-data-science"]` | Plugins to install        |

**Example with custom configuration:**

```bash
NEO4J_IMAGE=neo4j:5.25.0-enterprise \
NEO4J_USERNAME=admin \
NEO4J_PASSWORD=secret \
go test -tags=integration ./test/integration/... -v
```

## Important

- Always use `t.Parallel()` for parallel execution
- Always include `test_id` in queries for isolation
- Test data is automatically tagged with unique IDs and cleaned up
