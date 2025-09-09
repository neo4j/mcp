# Neo4j MCP Development Guide

Development documentation for the Neo4j MCP server.

## Testing with Mockgen

This project uses interface-based dependency injection with auto-generated mocks for testing without requiring a real Neo4j database.

### Setup & Usage

1. **Install mockgen**:

   ```bash
   go install go.uber.org/mock/mockgen@latest
   export PATH=$PATH:$(go env GOPATH)/bin
   ```

2. **Generate mocks** after interface changes:

   ```bash
   cd internal/database && go generate
   ```

3. **Run tests**:
   ```bash
   go test ./... -cover
   ```

### Testing Example

```go
func TestMyFunction(t *testing.T) {
    ctrl := gomock.NewController(t)
    defer ctrl.Finish()

    mockDB := mocks.NewMockDatabaseService(ctrl)

    mockDB.EXPECT().
        ExecuteReadQuery(gomock.Any(), "MATCH (n) RETURN n", gomock.Nil(), "neo4j").
        Return([]*neo4j.Record{}, nil)

    // Use mockDB in your test...
}
```

See `internal/tools/get_schema_handler_gomock_test.go` for complete examples.

## Project Structure

```
cmd/neo4j-mcp/         # Main application
internal/
├── database/          # Database interfaces & generated mocks
├── server/            # MCP server setup
└── tools/             # MCP tool handlers
```

```

```
