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

The MCP server supports two transport modes: **STDIO** (default) and **HTTP**. Required environment variables differ based on the mode.

### STDIO Mode (Default)

**Required variables:**

```bash
export NEO4J_URI="bolt://localhost:7687"
export NEO4J_USERNAME="neo4j"
export NEO4J_PASSWORD="password"
```

### HTTP Mode

**Required variables:**

```bash
export NEO4J_URI="bolt://localhost:7687"
export NEO4J_MCP_TRANSPORT="http"
```

**Note:** In HTTP mode, do NOT set `NEO4J_USERNAME` or `NEO4J_PASSWORD`. Credentials come from per-request Basic Auth headers.

### Optional Variables (Both Modes)

```bash
export NEO4J_DATABASE="neo4j"          # Default: neo4j
export NEO4J_READ_ONLY="false"         # Default: false (set to "true" to disable write tools)
export NEO4J_TELEMETRY="true"          # Default: true
export NEO4J_LOG_LEVEL="info"          # Default: info (debug, info, notice, warning, error, critical, alert, emergency)
export NEO4J_LOG_FORMAT="text"         # Default: text (text or json)
export NEO4J_SCHEMA_SAMPLE_SIZE="100"  # Default: 100 (number of nodes to sample for schema inference)

# HTTP mode specific (ignored in STDIO mode)
export NEO4J_MCP_HTTP_HOST="127.0.0.1" # Default: 127.0.0.1
export NEO4J_MCP_HTTP_PORT="80"        # Default: 80
export NEO4J_MCP_HTTP_ALLOWED_ORIGINS="*" # Default: empty (no CORS)
```

**Note:** Make sure your local Neo4j instance is running with the correct credentials before testing.

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

See `internal/tools/cypher/get_schema_handler_test.go` for a fuller pattern.

## Testing using the @modelcontextprotocol/inspector:

The Neo4j MCP capabilities can be tested using the `@modelcontextprotocol/inspector`:

```bash
npx @modelcontextprotocol/inspector go run ./cmd/neo4j-mcp
```

## Testing HTTP Mode

### Unit Tests

HTTP mode has comprehensive unit tests:

```bash
# Test middleware (CORS, Basic Auth, logging)
go test ./internal/server -v -run ".*Middleware.*"

# Test HTTP server configuration
go test ./internal/server -v -run ".*HTTP.*"

# Test database service with transport modes
go test ./internal/database -v

# Run all tests with coverage
go test ./... -cover
```

### Manual Testing

Start the server in HTTP mode:

```bash
# Set up environment
export NEO4J_URI="bolt://localhost:7687"
export NEO4J_MCP_TRANSPORT="http"

# Run server
go run ./cmd/neo4j-mcp
```

Test with curl:

```bash
# List available tools
curl -X POST http://localhost:80/mcp \
  -u "neo4j:password" \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}'

# Get Neo4j schema
curl -X POST http://localhost:80/mcp \
  -u "neo4j:password" \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "id": 2,
    "method": "tools/call",
    "params": {
      "name": "get-schema",
      "arguments": {}
    }
  }'

# Test authentication (should return 401)
curl -X POST http://localhost:80/mcp \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":3,"method":"tools/list"}'

# Test CORS (if configured)
curl -X OPTIONS http://localhost:80/mcp \
  -H "Origin: http://localhost:3000" \
  -H "Access-Control-Request-Method: POST"

# Test multi-user/multi-tenant (different credentials per request)
curl -X POST http://localhost:80/mcp \
  -u "userA:passwordA" \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":4,"method":"tools/list","params":{}}'

curl -X POST http://localhost:80/mcp \
  -u "userB:passwordB" \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":5,"method":"tools/list","params":{}}'
```

## TLS/HTTPS Configuration

For detailed instructions on generating certificates and testing TLS configurations, see the **[TLS Setup Guide](docs/TLS_SETUP.md)**.

This guide includes:
- Self-signed certificate generation for testing
- Testing TLS with curl and openssl
- TLS verification commands
- Production considerations (using Let's Encrypt certificates)

## MCP Error Handling

MCP error handling follows a specific pattern that differs from standard Go error handling. According to the [MCP specification](https://modelcontextprotocol.io/specification/2025-06-18/server/tools#error-handling), tool handlers should communicate errors through the tool result structure rather than returning Go errors directly.

### When to use MCP tool result errors vs direct Go errors:

- **Use MCP tool result errors** (`NewToolResultError`) for:

  - Business logic errors (invalid input, database constraints, etc.)
  - Operational errors that the client should handle gracefully
  - Any error that represents a meaningful response to the client

- **Return Go errors directly** for:
  - System-level failures (out of memory, network failures)
  - Programming errors that indicate bugs in the server implementation
  - Cases where the server cannot continue processing

### Recommended MCP Tool Handler error handling pattern:

When implementing MCP tool handlers, use the `mcp.NewToolResultError` helper function for cleaner error handling:

```go
func MyToolHandler(deps *ToolDependencies) mcp.ToolHandler {
    return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
        // Bind and validate arguments
        var args MyToolInput
        if err := request.BindArguments(&args); err != nil {
            return mcp.NewToolResultError("Invalid arguments: " + err.Error()), nil
        }

        // Business logic validation
        if args.SomeField == "" {
            return mcp.NewToolResultError("SomeField is required"), nil
        }

        // Execute operation
        result, err := someOperation(ctx, args)
        if err != nil {
            // Use MCP error for business/operational errors
            return mcp.NewToolResultError("Operation failed: " + err.Error()), nil
        }

        // Success case
        return mcp.NewToolResultText(result), nil
    }
}
```

**Note:** Always return `nil` as the second parameter when using `NewToolResultError`, as the error information is embedded within the `CallToolResult` structure.

## Adding New MCP Tools

1. **Define tool specifications** in `internal/tools/`:

   ```go
   func NewMyToolSpec() mcp.Tool {
       return mcp.NewTool("my-tool",
           mcp.WithDescription("Tool description"),
           mcp.WithInputSchema[MyToolInput](),
           mcp.WithReadOnlyHintAnnotation(true), // This flag will be used filter tools for the read-only mode.
       )
   }
   ```

   **Note:** WithReadOnlyHintAnnotation marks a tool with a read-only hint is used for filtering.
   When set to true, the tool will be considered read-only and included when selecting
   tools for read-only mode. If the annotation is not present or set to false,
   the tool is treated as a write-capable tool (i.e., not considered read-only).

2. **Implement tool handler**:

   ```go
   func NewMyToolHandler(deps *ToolDependencies) mcp.ToolHandler {
       return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
           // Implementation
       }
   }
   ```

3. **Register in tool_register.go, in the right section (cypher/GDS/etc...)**:

   ```go
   {
   		category: cypherCategory,
   		definition: server.ServerTool{
   			Tool:    cypher.GetSchemaSpec(),
   			Handler: cypher.GetSchemaHandler(deps),
   		},
   		readonly: true,
   },
   ```

4. **Write tests** with mocked dependencies

### Database Interface Extensions

When adding new database operations:

1. **Extend the interface** in `internal/database/interfaces.go`
2. **Implement in service** in `internal/database/service.go`
3. **Regenerate mocks**: `go generate ./...`
4. **Update tests** to use new mock methods

### Quick Fixes

- Mock generation fails → ensure `mockgen` on PATH.
- Tests failing unexpectedly → regenerate mocks, verify env vars, rerun full test suite.
- Dependency/build issues → `go mod tidy`.

## Update MCPB Bundle (for Claude Desktop)

If your changes impact the end-user configuration (e.g., adding new environment variables or modifying tool definitions), you must update the `manifest.json` file. This ensures that integrations like Claude Desktop are aware of the new server configuration.

For more information refer to the dedicated guide: [the MCPB build documentation](docs/BUILD_MCPB.md).

### Getting Help

- Check existing [GitHub Issues](https://github.com/neo4j/mcp/issues)
- Ask questions in pull request discussions
- Reach out to maintainers for complex architectural questions

Thank you for contributing to making Neo4j MCP better!
