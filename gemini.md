# GEMINI.md

This file provides guidance to Gemini Code (gemini.google.com/code) when working with code in this repository.

## Essential Commands

### Development
```bash
# Install dependencies
go mod download

# Build binary
go build -C cmd/neo4j-mcp -o ../../bin/

# Run from source
go run ./cmd/neo4j-mcp

# Install binary to GOPATH
go install -C cmd/neo4j-mcp
```

### Testing
```bash
# Run all tests with coverage
go test ./... -cover

# Run tests for specific package (verbose)
go test ./internal/tools -v

# Test single test
go test -run TestFunctionName ./internal/package
```

### Mock Generation
```bash
# Regenerate mocks after changing interfaces
cd internal/database && go generate

# Install mockgen if not available
go install go.uber.org/mock/mockgen@latest
export PATH="$PATH:$(go env GOPATH)/bin"
```

### Dependency Management
```bash
# Check for upgrades
go list -u -m all

# Upgrade all dependencies
go get -u all
```

## Environment Variables (Required for Testing)

```bash
export NEO4J_URI="bolt://localhost:7687"
export NEO4J_USERNAME="neo4j"
export NEO4J_PASSWORD="password"
export NEO4J_DATABASE="neo4j"
```

## Architecture Overview

This is a Neo4j Model Context Protocol (MCP) server that provides two main tools:
- `get-schema`: Introspects Neo4j database schema (labels, relationships, properties)
- `run-cypher`: Executes arbitrary Cypher queries

### Key Components

**Entry Point**: `cmd/neo4j-mcp/main.go`
- Loads configuration from environment variables
- Creates and starts the MCP server

**Server Layer**: `internal/server/server.go`
- `Neo4jMCPServer` struct manages the MCP server lifecycle
- Initializes Neo4j driver connection
- Registers available tools

**Database Layer**: `internal/database/`
- `interfaces.go`: Defines abstractions for Neo4j operations
- `service.go`: Implements DatabaseService interface
- Uses interface-based dependency injection for testability
- `mocks/`: Auto-generated mocks for testing

**Tools Layer**: `internal/tools/`
- `tool_register.go`: Central registration of all available tools
- `get_schema_*`: Schema introspection tool implementation
- `run_cypher_*`: Cypher execution tool implementation
- Each tool has spec, handler, and test files

**Configuration**: `internal/config/`
- Loads Neo4j connection details from environment variables

### MCP Error Handling Pattern

MCP tools return errors differently than standard Go patterns:

```go
// Instead of returning Go errors directly
if err != nil {
    return &mcp.CallToolResult{
        Content: []mcp.Content{
            mcp.NewTextContent(fmt.Sprintf("Error: %v", err)),
        },
        IsError: &[]bool{true}[0],
    }, nil // Note: return nil as the second parameter
}
```

### Adding New Tools

1. Define tool spec in `internal/tools/new_tool_spec.go`
2. Implement handler in `internal/tools/new_tool_handler.go`
3. Add tests in `internal/tools/new_tool_handler_test.go`
4. Register in `tool_register.go`
5. If adding database operations, extend `internal/database/interfaces.go` and regenerate mocks

### Testing with MCP Inspector

```bash
npx @modelcontextprotocol/inspector go run ./cmd/neo4j-mcp
```