package database

import (
	"context"
	"fmt"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

// ExecuteReadQuery executes a read-only Cypher query and returns raw records
func ExecuteReadQuery(ctx context.Context, driver *neo4j.DriverWithContext, cypher string, params map[string]any, database string) ([]*neo4j.Record, error) {
	res, err := neo4j.ExecuteQuery(ctx, *driver, cypher, params, neo4j.EagerResultTransformer, neo4j.ExecuteQueryWithDatabase(database), neo4j.ExecuteQueryWithReadersRouting())

	if err != nil {
		return nil, fmt.Errorf("failed to execute read query: %w", err)
	}

	return res.Records, nil
}

// ExecuteWriteQuery executes a write-only Cypher query and returns raw records
func ExecuteWriteQuery(ctx context.Context, driver *neo4j.DriverWithContext, cypher string, params map[string]any, database string) ([]*neo4j.Record, error) {
	res, err := neo4j.ExecuteQuery(ctx, *driver, cypher, params, neo4j.EagerResultTransformer, neo4j.ExecuteQueryWithDatabase(database), neo4j.ExecuteQueryWithWritersRouting())

	if err != nil {
		return nil, fmt.Errorf("failed to execute write query: %w", err)
	}

	return res.Records, nil
}
