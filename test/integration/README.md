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

## Important

- Always use `t.Parallel()` for parallel execution
- Always include `test_id` in queries for isolation
- Test data is automatically tagged with unique IDs and cleaned up
