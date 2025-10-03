package database

//go:generate mockgen -destination=mocks/mock_database.go -package=mocks github.com/neo4j/mcp/internal/database DatabaseService,DriverWithContext

import (
	"context"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

// Driver is a minimal interface wrapping neo4j.DriverWithContext for testability
type DriverWithContext interface {
	neo4j.DriverWithContext // Embedding the original interface to include all its methods
	// Additional methods can be added here if needed (for example, for mocking purposes and to use in tests)
	VerifyConnectivity(ctx context.Context) error
	Close(ctx context.Context) error
}

// QueryExecutor defines the interface for executing Neo4j queries
type QueryExecutor interface {
	// ExecuteReadQuery executes a read-only Cypher query and returns raw records
	ExecuteReadQuery(ctx context.Context, cypher string, params map[string]any, database string) ([]*neo4j.Record, error)

	// ExecuteWriteQuery executes a write-only Cypher query and returns raw records
	ExecuteWriteQuery(ctx context.Context, cypher string, params map[string]any, database string) ([]*neo4j.Record, error)
}

// RecordFormatter defines the interface for formatting Neo4j records
type RecordFormatter interface {
	// Neo4jRecordsToJSON converts Neo4j records to JSON string
	Neo4jRecordsToJSON(records []*neo4j.Record) (string, error)
}

// DatabaseService combines query execution and record formatting
type DatabaseService interface {
	QueryExecutor
	RecordFormatter
}
