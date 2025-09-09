# Database Interface Testing

This guide covers the database interfaces and their generated mocks for testing.

## Interface Structure

The database layer is organized around clean interfaces in `internal/database/`:

- `interfaces.go` - Defines the database interfaces:

  - `QueryExecutor` - Interface for executing Neo4j queries
  - `RecordFormatter` - Interface for formatting Neo4j records to JSON
  - `DatabaseService` - Combined interface that includes both query execution and record formatting

- `service.go` - Concrete implementation of the interfaces using Neo4j driver
- `mock.go` - Manual mock implementation (for reference/simple testing)

### Generating Mocks

To regenerate the mocks after interface changes:

```bash
cd internal/database
go generate
```

This will run the `//go:generate` directive in `interfaces.go` which executes:

```bash
mockgen -source=interfaces.go -destination=mocks/mock_database.go -package=mocks
```

### Prerequisites

Make sure `mockgen` is installed and in your PATH:

```bash
go install go.uber.org/mock/mockgen@latest
export PATH=$PATH:$(go env GOPATH)/bin
```

## Using the Generated Mocks

### Example with gomock

```go
func TestMyFunction(t *testing.T) {
    // Create mock controller
    ctrl := gomock.NewController(t)
    defer ctrl.Finish()

    // Create mock database service
    mockDB := mocks.NewMockDatabaseService(ctrl)

    // Set expectations
    mockDB.EXPECT().
        ExecuteReadQuery(gomock.Any(), "MATCH (n) RETURN n", gomock.Nil(), "neo4j").
        Return([]*neo4j.Record{}, nil).
        Times(1)

    mockDB.EXPECT().
        Neo4jRecordsToJSON(gomock.Any()).
        Return(`{"result": "test"}`, nil).
        Times(1)

    // Use the mock in your test
    result := MyFunction(mockDB)

    // Verify results
    // ... assertions ...
}
```

## Testing

The interface-based design enables comprehensive testing without requiring a real Neo4j database connection. All code paths can be tested including input validation, error handling, and success scenarios.
