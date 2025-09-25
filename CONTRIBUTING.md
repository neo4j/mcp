# Contributing to Neo4j MCP

Thank you for your interest in contributing to the Neo4j MCP server! This document provides guidelines and information for contributors.

If you're an external contributor you must sign the [https://neo4j.com/developer/contributing-code/#sign-cla](https://neo4j.com/developer/contributing-code/#sign-cla)

## Code of Conduct

Please read and follow these guidelines to ensure a welcoming environment for everyone.

## Prerequisites

- Go 1.25+ (see `go.mod`)
- A Neo4j instance with APOC plugin installed.

## Clone the repository (forks are currently disabled)

```bash
git clone git@github.com:neo4j/mcp.git && cd mcp
```

## Install Dependencies

```bash
# Install Go dependencies
go mod download

# Install mock generator (only if you will change interfaces, as the generated mocks depend on the interface definitions)
go install go.uber.org/mock/mockgen@latest
export PATH="$PATH:$(go env GOPATH)/bin"
```

## Environment Variables

The MCP server requires certain environment variables to connect to a Neo4j instance.
Defaults are provided for local development.
For local testing, make sure to set these environment variables (your local Neo4j instance must be running and it might require different credentials):

```bash
export NEO4J_URI="bolt://localhost:7687"
export NEO4J_USERNAME="neo4j"
export NEO4J_PASSWORD="password"
export NEO4J_DATABASE="neo4j"
```

## Build / Test / Run

```bash
# Tests (coverage)
go test ./... -cover

# Verbose / single package
go test ./internal/tools -v

# Build binary
go build -C cmd/neo4j-mcp -o ../../bin/

# Run from source
go run ./cmd/neo4j-mcp

# Optional: install (should be run from repo root)
go install -C cmd/neo4j-mcp
```

## Mocks

We rely on interface-based dependency injection plus generated mocks (gomock) so tests run without a live Neo4j instance.

Regenerate mocks ONLY after changing interfaces (e.g. `internal/database/interfaces.go`):

```bash
cd internal/database && go generate
```

Minimal gomock example:

```go
func TestMyFunction(t *testing.T) {
    ctrl := gomock.NewController(t)
    defer ctrl.Finish()

    mockDB := mocks.NewMockDatabaseService(ctrl)
    mockDB.EXPECT().
        ExecuteReadQuery(gomock.Any(), "MATCH (n) RETURN n", gomock.Nil(), "neo4j").
        Return([]*neo4j.Record{}, nil)

    // Use mockDB in your test ...
}
```

See `internal/tools/get_schema_handler_test.go` for a fuller pattern.

## Testing using the @modelcontextprotocol/inspector:

The Neo4j MCP capabilities can be tested using the `@modelcontextprotocol/inspector`:

```bash
npx @modelcontextprotocol/inspector go run ./cmd/neo4j-mcp
```

## MCP Error Handling

**Note:** MCP error handling differs from standard Go patterns. Instead of returning Go errors directly, we wrap error information inside a `CallToolResult` with the `IsError` flag set to true to properly communicate MCP-related errors to clients.

**MCP Tool Handler error handling pattern:**
When implementing MCP tool handlers, return errors using the MCP tool result structure:

```go
func MyToolHandler(deps *ToolDependencies) mcp.ToolHandler {
    return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
        // ... implementation ...

        if err != nil {
            return &mcp.CallToolResult{
                Content: []mcp.Content{
                    mcp.NewTextContent(fmt.Sprintf("Error: %v", err)),
                },
                IsError: &[]bool{true}[0],
            }, nil // Note: return nil as the second parameter, error info is in the result
        }

        // Success case
        return &mcp.CallToolResult{
            Content: []mcp.Content{
                mcp.NewTextContent("Success result"),
            },
        }, nil
    }
}
```

## Adding New MCP Tools

1. **Define tool specifications** in `internal/tools/`:

   ```go
   func NewMyToolSpec() mcp.Tool {
       return mcp.NewTool("my-tool",
           mcp.WithDescription("Tool description"),
           mcp.WithInputSchema[MyToolInput](),
       )
   }
   ```

2. **Implement tool handler**:

   ```go
   func NewMyToolHandler(deps *ToolDependencies) mcp.ToolHandler {
       return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
           // Implementation
       }
   }
   ```

3. **Register in tool_register.go**:

   ```go
   {
       Tool:    NewMyToolSpec(),
       Handler: NewMyToolHandler(deps),
   },
   ```

4. **Write tests** with mocked dependencies

### Database Interface Extensions

When adding new database operations:

1. **Extend the interface** in `internal/database/interfaces.go`
2. **Implement in service** in `internal/database/service.go`
3. **Regenerate mocks**: `cd internal/database && go generate`
4. **Update tests** to use new mock methods

### Quick Fixes

- Mock generation fails → ensure `mockgen` on PATH.
- Tests failing unexpectedly → regenerate mocks, verify env vars, rerun full test suite.
- Dependency/build issues → `go mod tidy`.

### Getting Help

- Check existing [GitHub Issues](https://github.com/neo4j/mcp/issues)
- Ask questions in pull request discussions
- Reach out to maintainers for complex architectural questions

Thank you for contributing to making Neo4j MCP better!
