# Contributing to Neo4j MCP

Thank you for your interest in contributing to the Neo4j MCP server! This document provides guidelines and information for contributors.

## Code of Conduct

This project follows the [Neo4j Community Guidelines](https://neo4j.com/developer/contributing/). Please read and follow these guidelines to ensure a welcoming environment for everyone.

## Project Status

Active development; not yet production‑hardened. Contributions that improve stability, correctness, performance, and ergonomics are welcome.

## Prerequisites

- Go 1.25+ (see `go.mod`)
- Git
- A Neo4j instance (4.x or 5.x; 5.x recommended)
- (Optional) APOC plugin for richer schema / procedures

## Getting Started

### Fork & Clone

1. Fork the repository (GitHub guide: https://docs.github.com/en/get-started/quickstart/fork-a-repo)
2. Clone your fork: `git clone https://github.com/<your-username>/mcp.git && cd mcp`
3. Add upstream remote (to sync later): `git remote add upstream https://github.com/neo4j/mcp.git`

### Install Dependencies

```bash
# Install Go dependencies
go mod download

# Install mock generator (only if you will change interfaces)
go install go.uber.org/mock/mockgen@latest
export PATH="$PATH:$(go env GOPATH)/bin"
```

### Local Development and Testing

```bash
# Build the binary
go build -C cmd/neo4j-mcp -o ../../bin/

# Run tests
go test ./... -v -cover

# Specific package
go test ./internal/tools -v

# Run directly without building
go run ./cmd/neo4j-mcp

# Install the binary (optional)
go install -C cmd/neo4j-mcp
```

### Environment Variables

For local testing, set these environment variables:

```bash
export NEO4J_URI="bolt://localhost:7687"
export NEO4J_USERNAME="neo4j"
export NEO4J_PASSWORD="password"
export NEO4J_DATABASE="neo4j"
```

## Mocks

We rely on interface-based dependency injection plus generated mocks (gomock) so tests run without a live Neo4j instance.

Regenerate mocks ONLY after changing interfaces (e.g. `internal/database/interfaces.go`):

```bash
cd internal/database && go generate
```

Example test using gomock:

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

See `internal/tools/get_schema_handler_gomock_test.go` for a fuller pattern.

Integration / manual check (optional):

```bash
npx @modelcontextprotocol/inspector go run ./cmd/neo4j-mcp
```

## Adding New MCP Tools

1. **Define tool specification** in `internal/tools/`:

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

## Code Standards

General Go style, logging, and performance patterns are intentionally omitted here to keep this concise. Follow idiomatic Go, keep interfaces small, wrap errors with context, and avoid leaking sensitive data. Open an issue if clarifications are needed.

## Documentation Updates

When you change user-visible behavior, update README, this file (if process changes), and any relevant examples.

## Troubleshooting Development Issues

### Common Development Problems

1. **Mock generation fails**: Ensure mockgen is installed and in PATH
2. **Tests fail**: Check if test Neo4j instance is running and accessible
3. **Build fails**: Run `go mod tidy` to clean up dependencies

### Getting Help

- Check existing [GitHub Issues](https://github.com/neo4j/mcp/issues)
- Review the [Troubleshooting Guide](docs/TROUBLESHOOTING.md)
- Ask questions in pull request discussions
- Reach out to maintainers for complex architectural questions

Thank you for contributing to making Neo4j MCP better!
