# Neo4j MCP Documentation

This directory contains detailed documentation for the Neo4j MCP server.

## Architecture Documentation

### Core Components

- [Database Interface Testing](database-interfaces.md) - Comprehensive guide to interface-based dependency injection, testing strategies, and mock generation

## Development Guides

### Testing

The project uses a comprehensive testing strategy with interface-based dependency injection:

#### Unit Testing with Mocks

- **Generated Mocks**: Use `gomock` for type-safe, auto-generated mocks from interfaces

#### Test Coverage

```bash
# Run tests with coverage report
go test ./... -cover

# Generate detailed coverage report
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out
```

#### Writing Tests

1. **Use interfaces**: All database operations go through `DatabaseService` interface
2. **Mock dependencies**: Use generated mocks for external dependencies
3. **Test all paths**: Success, error, and edge cases
4. **Validate inputs**: Test parameter validation and error handling

#### Mock Generation Workflow

```bash
# 1. Install mockgen
go install go.uber.org/mock/mockgen@latest

# make sure it's in your PATH
export PATH=$PATH:$(go env GOPATH)/bin

# 2. Generate mocks after interface changes
cd internal/database && go generate

# 3. Run tests to verify mocks work
go test ./internal/tools/ -v
```

## Getting Started

1. Read the main [README](../README.md) for setup instructions
2. Explore [Database Interface Testing](database-interfaces.md) to understand the testing architecture
3. Check out the example test files in `internal/tools/` for practical examples

## Contributing

When adding new features:

1. Follow the interface-based design patterns
2. Add comprehensive tests (both unit and integration where appropriate)
3. Use `go generate` to update mocks when interfaces change
4. Update documentation as needed

## Quick Reference

### Common Commands

```bash
# Generate mocks
cd internal/database && go generate

# Run tests with coverage
go test ./... -cover

# Run specific test patterns
go test ./internal/tools/ -run="TestGetSchemaHandler"
```

### Project Structure

```
cmd/
└── neo4j-mcp/         # Main application entry point

internal/
├── config/            # Configuration management
├── database/          # Database interfaces and implementations
│   ├── interfaces.go  # Interface definitions
│   ├── service.go     # Neo4j implementation
│   ├── execute_query.go
│   ├── neo4_records_to_json.go
│   └── mocks/         # Generated mocks (auto-generated)
├── server/            # MCP server setup
└── tools/             # MCP tool handlers
    ├── *_handler.go   # Tool implementations
    ├── *_spec.go      # Tool specifications
    └── *_test.go      # Tests

docs/                  # Documentation
```
