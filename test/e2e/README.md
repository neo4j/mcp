# End-to-End (E2E) Tests

End-to-end tests for the Neo4j MCP server that test the complete server lifecycle including compilation, MCP protocol communication, and tool execution using a shared Neo4j container (includes APOC + GDS).

## Quick Start

```go
func TestMyE2EFeature(t *testing.T) {
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

    // Create test context for database operations (auto-isolation + cleanup)
    tc := helpers.NewE2ETestContext(t, dbs.GetDriver())

    // Seed test data (automatically isolated with unique labels and cleaned up)
    personLabel, err := tc.SeedNode("Person", map[string]any{"name": "Alice"})
    if err != nil {
        t.Fatalf("failed to seed data: %v", err)
    }

    // Call MCP tool
    callToolRequest := mcp.CallToolRequest{
        Params: mcp.CallToolParams{
            Name: "read-cypher",
            Arguments: map[string]any{
                "query":  "MATCH (p:" + personLabel.String() + " {name: $name}) RETURN p",
                "params": map[string]any{"name": "Alice"},
            },
        },
    }

    callToolResponse, err := mcpClient.CallTool(ctx, callToolRequest)
    if err != nil {
        t.Fatalf("failed to call tool: %v", err)
    }

    // Verify response
    textContent, ok := mcp.AsTextContent(callToolResponse.Content[0])
    if !ok {
        t.Fatalf("expected TextContent, got %T", callToolResponse.Content[0])
    }

    // Parse and assert
    tc.AssertJSONListContainsObject(textContent.Text, map[string]any{
        "p": map[string]any{"name": "Alice"},
    })
}
```

## Key Helpers

**E2ETestContext:**

- `helpers.NewE2ETestContext(t, dbs.GetDriver())` - Auto-isolation + cleanup for database operations
- `SeedNode(label, props)` - Create test data with unique label, returns `(UniqueLabel, error)`
- `GetUniqueLabel(label)` - Get a unique label for creating nodes manually
- `AssertJSONListContainsObject(responseBody, expectedItem)` - Assert JSON response contains expected object

**MCP Protocol Helpers:**

- `helpers.BuildInitializeRequest()` - Create standard MCP initialization request
- `client.NewStdioMCPClient(server, env, args...)` - Create MCP client for server communication
- `mcpClient.Initialize(ctx, request)` - Initialize MCP server
- `mcpClient.CallTool(ctx, request)` - Call MCP tools
- `mcp.AsTextContent(content)` - Extract text content from MCP responses

**Global Variables:**

- `server` - Path to compiled MCP server binary (set up in TestMain)
- `dbs` - Shared database service for Neo4j container management

## Test Structure

E2E tests follow this pattern:

1. **Server Setup** (done in `TestMain`):

   - Compile the MCP server binary using `helpers.BuildServer()`
   - Start Neo4j container with `dbs.Start(ctx)`
   - Store server binary path in global `server` variable

2. **Per-Test Setup**:

   - Create MCP client with database connection args
   - Initialize MCP server via protocol
   - Set up cleanup to close client

3. **Test Execution**:

   - Create `E2ETestContext` for database isolation
   - Seed test data using unique labels
   - Execute MCP tool calls through the client
   - Assert on responses and database state

4. **Cleanup** (automatic):
   - Database cleanup via unique label deletion
   - MCP client closure
   - Server binary cleanup (in TestMain)

## Running Tests

```bash
go test -tags=e2e ./test/e2e/... -v                    # All e2e tests
go test -tags=e2e ./test/e2e/... -run MyFeature        # Specific test
go test -tags=e2e ./test/e2e/... -race                 # With race detection
go test -tags=e2e ./test/e2e/... -timeout 10m          # Extended timeout
```

## Configuration

E2E tests use the same environment variables as integration tests to configure Neo4j connection:

### Container vs. External Database

Use `USE_CONTAINER` to control whether tests start a Neo4j container or connect to an external database:

| Environment Variable | Default | Description                                                                  |
| -------------------- | ------- | ---------------------------------------------------------------------------- |
| `USE_CONTAINER`      | `true`  | When `true`, starts a Docker container; when `false`, uses external database |

**Example with container (default):**

```bash
NEO4J_IMAGE=neo4j:5-community \
NEO4J_USERNAME=admin \
NEO4J_PASSWORD=secret \
go test -tags=e2e ./test/e2e/... -v
```

**Example with external database:**

```bash
USE_CONTAINER=false \
NEO4J_URI=bolt://neo4j.example.com:7687 \
NEO4J_USERNAME=admin \
NEO4J_PASSWORD=secret \
go test -tags=e2e ./test/e2e/... -v
```

## Important Notes

- Always use `t.Parallel()` for parallel execution
- Always use the `UniqueLabel` returned by `SeedNode()` or `GetUniqueLabel()` in your queries for isolation
- Test data is automatically tagged with unique labels and cleaned up after each test
- Server binary is compiled once in `TestMain`, stored in a Temporary Directory and reused across all tests
- MCP client connections are per-test and must be properly closed
- Import the helpers package: `"github.com/neo4j/mcp/test/e2e/helpers"`
