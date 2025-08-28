package database

import (
	"context"
	"fmt"
	"os"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

// ExecuteReadQuery executes a read-only Cypher query and returns raw records
func ExecuteReadQuery(ctx context.Context, driver *neo4j.DriverWithContext, cypher string, params map[string]any, database string) ([]*neo4j.Record, error) {
	res, err := neo4j.ExecuteQuery(ctx, *driver, cypher, params, neo4j.EagerResultTransformer, neo4j.ExecuteQueryWithDatabase(database), neo4j.ExecuteQueryWithReadersRouting())

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error while executing Cypher: %v\n", err)
		return nil, err
	}

	return res.Records, nil
}

// ExecuteWriteQuery executes a write-only Cypher query and returns raw records
func ExecuteWriteQuery(ctx context.Context, driver *neo4j.DriverWithContext, cypher string, params map[string]any, database string) ([]*neo4j.Record, error) {
	res, err := neo4j.ExecuteQuery(ctx, *driver, cypher, params, neo4j.EagerResultTransformer, neo4j.ExecuteQueryWithDatabase(database))

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error while executing Cypher: %v\n", err)
		return nil, err
	}

	return res.Records, nil
}
