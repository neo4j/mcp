package database

//go:generate mockgen -destination=mocks/mock_database.go -package=mocks github.com/neo4j/mcp/internal/database Service

import (
	"context"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

// QueryExecutor defines the interface for executing Neo4j queries
type QueryExecutor interface {
	// ExecuteReadQuery executes a read-only Cypher query and returns raw records
	ExecuteReadQuery(ctx context.Context, cypher string, params map[string]any) ([]*neo4j.Record, error)

	// ExecuteWriteQuery executes a write-only Cypher query and returns raw records
	ExecuteWriteQuery(ctx context.Context, cypher string, params map[string]any) ([]*neo4j.Record, error)

	// GetQueryType prefixes the provided query with EXPLAIN and returns the query type (e.g. 'r' for read, 'w' for write, 'rw' etc.)
	// This allows read-only tools to determine if a query is safe to run in read-only context.
	GetQueryType(ctx context.Context, cypher string, params map[string]any) (neo4j.StatementType, error)
}

// RecordFormatter defines the interface for formatting Neo4j records
type RecordFormatter interface {
	// Neo4jRecordsToJSON converts Neo4j records to JSON string
	Neo4jRecordsToJSON(records []*neo4j.Record) (string, error)
}

// Service combines query execution and record formatting
type Service interface {
	QueryExecutor
	RecordFormatter
}
