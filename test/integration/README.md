# Integration Tests

Integration tests for the Neo4j MCP server using a shared Neo4j container (includes APOC + GDS).

## Quick Start

```go
func TestMyFeature(t *testing.T) {
    t.Parallel()
    tc := helpers.NewTestContext(t, container_runner.GetDriver())

    // Seed test data (automatically isolated with unique labels and cleaned up)
    personLabel, err := tc.SeedNode("Person", map[string]any{"name": "Alice"})
    if err != nil {
        t.Fatalf("failed to seed data: %v", err)
    }

    // Call tool
    handler := cypher.ReadCypherHandler(tc.Deps)
    res := tc.CallTool(handler, map[string]any{
        "query":  "MATCH (p:" + personLabel + " {name: $name}) RETURN p",
        "params": map[string]any{"name": "Alice"},
    })

    // Parse and assert
    var records []map[string]any
    tc.ParseJSONResponse(res, &records)

    person := records[0]["p"].(map[string]any)
    helpers.AssertNodeProperties(t, person, map[string]any{"name": "Alice"})
    helpers.AssertNodeHasLabel(t, person, personLabel)
}
```

## Key Helpers

**TestContext:**

- `helpers.NewTestContext(t, dbs.GetDriver())` - Auto-isolation + cleanup
- `SeedNode(label, props)` - Create test data with unique label, returns `(UniqueLabel, error)`
- `GetUniqueLabel(label)` - Get a unique label for creating nodes manually
- `CallTool(handler, args)` - Invoke MCP tool
- `ParseJSONResponse(res, &v)` - Parse response
- `VerifyNodeInDB(label, props)` - Check DB state

**Assertions:**

- `helpers.AssertNodeProperties(t, node, props)`
- `helpers.AssertNodeHasLabel(t, node, label)`
- `helpers.AssertSchemaHasNodeType(t, schema, label, props)`

## Running Tests

```bash
go test -tags=integration ./test/integration/... -v              # All tests
go test -tags=integration ./test/integration/... -run MyFeature  # Specific test
go test -tags=integration ./test/integration/... -race           # With race detection
```

## Configuration

The integration tests use environment variables to configure how they connect to Neo4j. All variables have sensible defaults.

### Container vs. External Database

Use `MCP_USE_CONTAINER_FOR_INTEGRATION_TESTS` to control whether tests start a Neo4j container or connect to an external database:

| Environment Variable                      | Default | Description                                                                  |
| ----------------------------------------- | ------- | ---------------------------------------------------------------------------- |
| `MCP_USE_CONTAINER_FOR_INTEGRATION_TESTS` | `true`  | When `true`, starts a Docker container; when `false`, uses external database |

**Example with container (default):**

```bash
NEO4J_IMAGE=neo4j:5-community \
NEO4J_USERNAME=admin \
NEO4J_PASSWORD=secret \
go test -tags=integration ./test/integration/... -v
```

**Example with external database:**

```bash
MCP_USE_CONTAINER_FOR_INTEGRATION_TESTS=false \
NEO4J_URI=bolt://neo4j.example.com:7687 \
NEO4J_USERNAME=admin \
NEO4J_PASSWORD=secret \
go test -tags=integration ./test/integration/... -v
```

## Important

- Always use `t.Parallel()` for parallel execution
- Always use the `UniqueLabel` returned by `SeedNode()` or `GetUniqueLabel()` in your queries for isolation
- Test data is automatically tagged with unique labels and cleaned up after each test
- Import the helpers package: `"github.com/neo4j/mcp/test/integration/helpers"`
