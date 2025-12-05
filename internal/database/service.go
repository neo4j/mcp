package database

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

// Neo4jService is the concrete implementation of DatabaseService
type Neo4jService struct {
	driver   neo4j.DriverWithContext
	database string
}

// NewNeo4jService creates a new Neo4jService instance
func NewNeo4jService(driver neo4j.DriverWithContext, database string) (*Neo4jService, error) {
	if driver == nil {
		return nil, fmt.Errorf("driver cannot be nil")
	}

	return &Neo4jService{
		driver:   driver,
		database: database,
	}, nil
}

// VerifyConnectivity checks the driver can establish a valid connection with a Neo4j instance;
func (s *Neo4jService) VerifyConnectivity(ctx context.Context) error {
	// Verify database connectivity
	if err := s.driver.VerifyConnectivity(ctx); err != nil {
		slog.Error("Failed to verify database connectivity", "error", err.Error())
		return err
	}
	return nil
}

// ExecuteReadQuery executes a read-only Cypher query and returns raw records
func (s *Neo4jService) ExecuteReadQuery(ctx context.Context, cypher string, params map[string]any) ([]*neo4j.Record, error) {
	res, err := neo4j.ExecuteQuery(ctx, s.driver, cypher, params, neo4j.EagerResultTransformer, neo4j.ExecuteQueryWithDatabase(s.database), neo4j.ExecuteQueryWithReadersRouting())
	if err != nil {
		wrappedErr := fmt.Errorf("failed to execute read query: %w", err)
		slog.Error("Error in ExecuteReadQuery", "error", wrappedErr)
		return nil, wrappedErr
	}

	return res.Records, nil
}

// ExecuteReadQueryWithAuth validates the query is read-only, then executes it using per-request credentials via impersonation.
// Uses Neo4j's impersonation feature to execute queries with different credentials without creating new drivers.
// Returns an error if the query is not read-only.
func (s *Neo4jService) ExecuteReadQueryWithAuth(ctx context.Context, username, password, cypher string, params map[string]any) ([]*neo4j.Record, error) {
	// Create auth token for impersonation
	queryAuth := neo4j.BasicAuth(username, password, "")

	// First, validate query type using EXPLAIN with the provided credentials
	explainedQuery := strings.Join([]string{"EXPLAIN", cypher}, " ")
	explainRes, err := neo4j.ExecuteQuery(ctx, s.driver, explainedQuery, params,
		neo4j.EagerResultTransformer,
		neo4j.ExecuteQueryWithDatabase(s.database),
		neo4j.ExecuteQueryWithAuthToken(queryAuth))
	if err != nil {
		wrappedErr := fmt.Errorf("failed to validate query type with per-request auth: %w", err)
		slog.Error("Error validating query type in ExecuteReadQueryWithAuth", "error", wrappedErr, "username", username)
		return nil, wrappedErr
	}

	if explainRes.Summary == nil {
		err := fmt.Errorf("no summary returned for explained query")
		slog.Error("Error in ExecuteReadQueryWithAuth", "error", err, "username", username)
		return nil, err
	}

	queryType := explainRes.Summary.StatementType()
	if queryType != neo4j.StatementTypeReadOnly {
		wrappedErr := fmt.Errorf("query is not read-only (type: %v)", queryType)
		slog.Error("Rejected non-read query in ExecuteReadQueryWithAuth", "error", wrappedErr, "username", username, "queryType", queryType)
		return nil, wrappedErr
	}

	// Query is validated as read-only, now execute it using impersonation
	res, err := neo4j.ExecuteQuery(ctx, s.driver, cypher, params,
		neo4j.EagerResultTransformer,
		neo4j.ExecuteQueryWithDatabase(s.database),
		neo4j.ExecuteQueryWithReadersRouting(),
		neo4j.ExecuteQueryWithAuthToken(queryAuth))
	if err != nil {
		wrappedErr := fmt.Errorf("failed to execute read query with per-request auth: %w", err)
		slog.Error("Error executing query in ExecuteReadQueryWithAuth", "error", wrappedErr, "username", username)
		return nil, wrappedErr
	}

	return res.Records, nil
}

// ExecuteWriteQuery executes a write-only Cypher query and returns raw records
func (s *Neo4jService) ExecuteWriteQuery(ctx context.Context, cypher string, params map[string]any) ([]*neo4j.Record, error) {
	res, err := neo4j.ExecuteQuery(ctx, s.driver, cypher, params, neo4j.EagerResultTransformer, neo4j.ExecuteQueryWithDatabase(s.database), neo4j.ExecuteQueryWithWritersRouting())
	if err != nil {
		wrappedErr := fmt.Errorf("failed to execute write query: %w", err)
		slog.Error("Error in ExecuteWriteQuery", "error", wrappedErr)
		return nil, wrappedErr
	}

	return res.Records, nil
}

// GetQueryType prefixes the provided query with EXPLAIN and returns the query type (e.g. 'r' for read, 'w' for write, 'rw' etc.)
// This allows read-only tools to determine if a query is safe to run in read-only context.
func (s *Neo4jService) GetQueryType(ctx context.Context, cypher string, params map[string]any) (neo4j.StatementType, error) {
	if s.driver == nil {
		err := fmt.Errorf("neo4j driver is not initialized")
		slog.Error("Error in GetQueryType", "error", err)
		return neo4j.StatementTypeUnknown, err
	}

	explainedQuery := strings.Join([]string{"EXPLAIN", cypher}, " ")
	res, err := neo4j.ExecuteQuery(ctx, s.driver, explainedQuery, params, neo4j.EagerResultTransformer, neo4j.ExecuteQueryWithDatabase(s.database))
	if err != nil {
		wrappedErr := fmt.Errorf("error during GetQueryType: %w", err)
		slog.Error("Error during GetQueryType", "error", wrappedErr)
		return neo4j.StatementTypeUnknown, wrappedErr
	}

	if res.Summary == nil {
		err := fmt.Errorf("error during GetQueryType: no summary returned for explained query")
		slog.Error("Error during GetQueryType", "error", err)
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
		slog.Error("Error in Neo4jRecordsToJSON", "error", wrappedErr)
		return "", wrappedErr
	}

	return string(formattedResponse), nil
}
