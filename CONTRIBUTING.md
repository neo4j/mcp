# Contributing to Neo4j MCP

Thank you for your interest in contributing to the Neo4j MCP server! This document provides guidelines and information for contributors.

## Code of Conduct

This project follows the [Neo4j Community Guidelines](https://neo4j.com/developer/contributing/). Please read and follow these guidelines to ensure a welcoming environment for everyone.

## Development Status

This project is currently under active development and is not ready for production use. We welcome contributions to help make it production-ready.

## Prerequisites for Development

- Go 1.19 or later
- Git
- Neo4j database instance for testing (4.0 or later recommended)
- APOC plugin installed in your test Neo4j instance

## Getting Started

### 1. Fork and Clone

```bash
# Fork the repository on GitHub, then clone your fork
git clone https://github.com/YOUR_USERNAME/mcp.git
cd mcp

# Add the original repository as upstream
git remote add upstream https://github.com/neo4j/mcp.git
```

### 2. Set up Development Environment

```bash
# Install Go dependencies
go mod download

# Install development tools
go install go.uber.org/mock/mockgen@latest
export PATH=$PATH:$(go env GOPATH)/bin

# Set up Go environment (if not already configured)
export GOPATH=$HOME/go
export GOBIN=$GOPATH/bin
export PATH=$PATH:$GOBIN
```

### 3. Build and Test

```bash
# Build the project
go build -C cmd/neo4j-mcp -o ../../bin/

# Run tests
go test ./... -v -cover

# Generate mocks (after interface changes)
cd internal/database && go generate
```

## Development Workflow

### Branch Naming

- `feature/description` - for new features
- `fix/description` - for bug fixes
- `docs/description` - for documentation updates
- `refactor/description` - for code refactoring

### Making Changes

1. **Create a branch** from `main`:

   ```bash
   git checkout main
   git pull upstream main
   git checkout -b feature/your-feature-name
   ```

2. **Make your changes** following our coding standards

3. **Write or update tests** for your changes

4. **Run tests** to ensure everything works:

   ```bash
   go test ./... -v -cover
   ```

5. **Generate mocks** if you modified interfaces:

   ```bash
   cd internal/database && go generate
   ```

6. **Commit your changes** with clear commit messages:
   ```bash
   git add .
   git commit -m "feat: add new schema introspection feature"
   ```

### Commit Message Format

We follow the [Conventional Commits](https://conventionalcommits.org/) specification:

- `feat:` - new features
- `fix:` - bug fixes
- `docs:` - documentation changes
- `test:` - adding or updating tests
- `refactor:` - code refactoring
- `style:` - code style changes
- `chore:` - maintenance tasks

Examples:

```
feat: add support for multiple databases
fix: handle connection timeout gracefully
docs: update installation instructions
test: add coverage for schema handler
```

## Testing

### Unit Tests

We use interface-based dependency injection with auto-generated mocks for testing without requiring a real Neo4j database.

```bash
# Run all tests
go test ./... -cover

# Run tests with verbose output
go test ./... -v -cover

# Run tests for a specific package
go test ./internal/tools -v
```

### Mock Generation

After modifying interfaces, regenerate mocks:

```bash
cd internal/database && go generate
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

### Integration Testing

For testing with a real Neo4j instance, use the MCP Inspector:

```bash
npx @modelcontextprotocol/inspector go run ./cmd/neo4j-mcp
```

This launches an interactive interface to test MCP server functionality.

## Code Standards

### Go Style

- Follow [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- Use `gofmt` to format your code
- Run `go vet` to check for common errors
- Use meaningful variable and function names
- Write godoc comments for exported functions and types

### Architecture Principles

This project follows a clean architecture pattern with dependency injection:

- **Interface-based design**: All external dependencies are abstracted behind interfaces
- **Dependency injection**: Dependencies are injected rather than created within components
- **Separation of concerns**: Clear boundaries between server setup, tool handling, and database operations
- **Testability**: Mock implementations enable unit testing without external dependencies

### Data Flow

1. **MCP Client Request** → 2. **Server** → 3. **Tool Handler** → 4. **Database Service** → 5. **Neo4j Database**

- Client sends JSON-RPC request over stdio
- Server routes to appropriate tool handler
- Handler validates input and calls database service
- Service executes Neo4j operations
- Results flow back through the same path

### Project Structure

```
cmd/neo4j-mcp/         # Main application entry point
internal/
├── config/            # Configuration management
├── database/          # Database interfaces & implementations
├── server/            # MCP server setup and initialization
└── tools/             # MCP tool handlers and specifications
docs/                  # Documentation files
```

### Interface Design

- Define interfaces in the package that uses them
- Keep interfaces small and focused
- Use dependency injection for testability
- Generate mocks for all interfaces using `//go:generate`

### Error Handling and Logging

**Standard error handling pattern:**
```go
result, err := deps.DBService.ExecuteQuery(ctx, query, params, database)
if err != nil {
    return nil, fmt.Errorf("failed to execute query: %w", err)
}
```

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

**Logging guidelines:**
- Use `log` package (writes to stderr for MCP compliance)
- Currently uses simple logging without levels - log important events and errors
- Include context in log messages
- Avoid logging sensitive information (credentials, data)

### Performance Considerations

- Use context cancellation for long-running operations
- Implement connection pooling in database service
- Consider query result streaming for large datasets
- Monitor memory usage with large graph responses

## Advanced Development

### Adding New MCP Tools

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

## Documentation

### Code Documentation

- Write clear godoc comments for all exported functions and types
- Include examples in godoc when helpful
- Document complex algorithms and business logic

### User Documentation

- Update relevant documentation when adding features
- Include examples for new functionality
- Update the troubleshooting guide for new error conditions

## Submitting Changes

### Pull Request Process

1. **Push your branch** to your fork:

   ```bash
   git push origin feature/your-feature-name
   ```

2. **Create a pull request** on GitHub with:

   - Clear title and description
   - Reference to any related issues
   - Screenshots/examples if relevant
   - Test coverage information

3. **Address review feedback** promptly and professionally

4. **Ensure CI passes** before requesting final review

### Pull Request Checklist

- [ ] Code follows project standards
- [ ] Tests are written and passing
- [ ] Documentation is updated
- [ ] Commit messages follow conventional format
- [ ] No breaking changes (or clearly documented)
- [ ] Mocks are regenerated if interfaces changed

## Building and Running

### Local Development

```bash
# Build the binary
go build -C cmd/neo4j-mcp -o ../../bin/

# Run directly without building
go run ./cmd/neo4j-mcp

# Install to GOBIN
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

## Troubleshooting Development Issues

### Common Development Problems

1. **Mock generation fails**: Ensure mockgen is installed and in PATH
2. **Tests fail**: Check if test Neo4j instance is running and accessible
3. **Build fails**: Run `go mod tidy` to clean up dependencies
4. **Import cycles**: Restructure packages to avoid circular dependencies

### Getting Help

- Check existing [GitHub Issues](https://github.com/neo4j/mcp/issues)
- Review the [Troubleshooting Guide](docs/TROUBLESHOOTING.md)
- Ask questions in pull request discussions
- Reach out to maintainers for complex architectural questions

## Release Process

(For maintainers)

1. Update version numbers
2. Update changelog
3. Create release branch
4. Tag release
5. Update documentation
6. Announce release

## Recognition

Contributors will be recognized in:

- GitHub contributors list
- Release notes for significant contributions
- Project documentation credits

Thank you for contributing to making Neo4j MCP better!
