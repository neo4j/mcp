package database

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

// Neo4jService is the concrete implementation of DatabaseService
type Neo4jService struct {
	driver *neo4j.DriverWithContext
}

// NewNeo4jService creates a new Neo4jService instance
func NewNeo4jService(driver *neo4j.DriverWithContext) Service {
	return &Neo4jService{
		driver: driver,
	}
}

// ExecuteReadQuery executes a read-only Cypher query and returns raw records
func (s *Neo4jService) ExecuteReadQuery(ctx context.Context, cypher string, params map[string]any, database string) ([]*neo4j.Record, error) {
	if s.driver == nil {
		err := fmt.Errorf("Neo4j driver is not initialized")
		log.Printf("Error in ExecuteReadQuery: %v", err)
		return nil, err
	}

	res, err := neo4j.ExecuteQuery(ctx, *s.driver, cypher, params, neo4j.EagerResultTransformer, neo4j.ExecuteQueryWithDatabase(database), neo4j.ExecuteQueryWithReadersRouting())

	if err != nil {
		wrappedErr := fmt.Errorf("failed to execute read query: %w", err)
		log.Printf("Error in ExecuteReadQuery: %v", wrappedErr)
		return nil, wrappedErr
	}

	return res.Records, nil
}

// ExecuteWriteQuery executes a write-only Cypher query and returns raw records
func (s *Neo4jService) ExecuteWriteQuery(ctx context.Context, cypher string, params map[string]any, database string) ([]*neo4j.Record, error) {
	if s.driver == nil {
		err := fmt.Errorf("Neo4j driver is not initialized")
		log.Printf("Error in ExecuteWriteQuery: %v", err)
		return nil, err
	}

	res, err := neo4j.ExecuteQuery(ctx, *s.driver, cypher, params, neo4j.EagerResultTransformer, neo4j.ExecuteQueryWithDatabase(database), neo4j.ExecuteQueryWithWritersRouting())

	if err != nil {
		wrappedErr := fmt.Errorf("failed to execute write query: %w", err)
		log.Printf("Error in ExecuteWriteQuery: %v", wrappedErr)
		return nil, wrappedErr
	}

	return res.Records, nil
}

// GetQueryType prefixes the provided query with EXPLAIN and returns the query type (e.g. 'r' for read, 'w' for write, 'rw' etc.)
// This allows read-only tools to determine if a query is safe to run in read-only context.
func (s *Neo4jService) GetQueryType(ctx context.Context, cypher string, params map[string]any, database string) (neo4j.StatementType, error) {
	if s.driver == nil {
		err := fmt.Errorf("neo4j driver is not initialized")
		log.Printf("Error in GetQueryType: %v", err)
		return neo4j.StatementTypeUnknown, err
	}

	explainedQuery := strings.Join([]string{"EXPLAIN", cypher}, " ")
	res, err := neo4j.ExecuteQuery(ctx, *s.driver, explainedQuery, params, neo4j.EagerResultTransformer, neo4j.ExecuteQueryWithDatabase(database))
	if err != nil {
		wrappedErr := fmt.Errorf("error during GetQueryType: %w", err)
		log.Printf("Error during GetQueryType: %v", wrappedErr)
		return neo4j.StatementTypeUnknown, wrappedErr
	}

	if res.Summary == nil {
		err := fmt.Errorf("error during GetQueryType: no summary returned for explained query")
		log.Printf("Error during GetQueryType: %v", err)
		return neo4j.StatementTypeUnknown, err
	}

	return res.Summary.StatementType(), nil

}

// Neo4jRecordsToJSON converts Neo4j records to JSON string
func (s *Neo4jService) Neo4jRecordsToJSON(records []*neo4j.Record) (string, error) {
	results := make([]map[string]any, 0)
	for _, record := range records {
		recordMap := record.AsMap()
		results = append(results, recordMap)
	}

	formattedResponse, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		wrappedErr := fmt.Errorf("failed to format records as JSON: %w", err)
		log.Printf("Error in Neo4jRecordsToJSON: %v", wrappedErr)
		return "", wrappedErr
	}

	formattedResponseStr := string(formattedResponse)

	return formattedResponseStr, nil
}
