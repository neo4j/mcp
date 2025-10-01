package database

//go:generate mockgen -source=interfaces.go -destination=mocks/mock_database.go -package=mocks

import (
	"context"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

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

// Service combines query execution and record formatting
type Service interface {
	QueryExecutor
	RecordFormatter
}
