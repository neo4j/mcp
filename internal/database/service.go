package database

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

// Neo4jService is the concrete implementation of DatabaseService
type Neo4jService struct {
	driver *neo4j.DriverWithContext
}

// NewNeo4jService creates a new Neo4jService instance
func NewNeo4jService(driver *neo4j.DriverWithContext) DatabaseService {
	return &Neo4jService{
		driver: driver,
	}
}

// ExecuteReadQuery executes a read-only Cypher query and returns raw records
func (s *Neo4jService) ExecuteReadQuery(ctx context.Context, cypher string, params map[string]any, database string) ([]*neo4j.Record, error) {
	if s.driver == nil {
		return nil, fmt.Errorf("Neo4j driver is not initialized")
	}

	res, err := neo4j.ExecuteQuery(ctx, *s.driver, cypher, params, neo4j.EagerResultTransformer, neo4j.ExecuteQueryWithDatabase(database), neo4j.ExecuteQueryWithReadersRouting())

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error while executing Cypher: %v\n", err)
		return nil, err
	}

	return res.Records, nil
}

// ExecuteWriteQuery executes a write-only Cypher query and returns raw records
func (s *Neo4jService) ExecuteWriteQuery(ctx context.Context, cypher string, params map[string]any, database string) ([]*neo4j.Record, error) {
	if s.driver == nil {
		return nil, fmt.Errorf("Neo4j driver is not initialized")
	}

	res, err := neo4j.ExecuteQuery(ctx, *s.driver, cypher, params, neo4j.EagerResultTransformer, neo4j.ExecuteQueryWithDatabase(database), neo4j.ExecuteQueryWithWritersRouting())

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error while executing Cypher: %v\n", err)
		return nil, err
	}

	return res.Records, nil
}

// Neo4jRecordsToJSON converts Neo4j records to JSON string
func (s *Neo4jService) Neo4jRecordsToJSON(records []*neo4j.Record) (string, error) {
	var results []map[string]any
	for _, record := range records {
		recordMap := record.AsMap()
		results = append(results, recordMap)
	}

	formattedResponse, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error formatting response as JSON: %v\n", err)
		return "", err
	}

	formattedResponseStr := string(formattedResponse)
	fmt.Fprintf(os.Stderr, "The formatted response: %s\n", formattedResponseStr)

	return formattedResponseStr, nil
}
