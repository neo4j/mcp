package database

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/neo4j/mcp/internal/auth"
	"github.com/neo4j/mcp/internal/config"
	"github.com/neo4j/neo4j-go-driver/v6/neo4j"
)

const appName string = "MCP4NEO4J"

// Neo4jService is the concrete implementation of DatabaseService
type Neo4jService struct {
	driver          neo4j.Driver
	database        string
	transportMode   string // Transport mode (stdio or http)
	neo4jMCPVersion string
}

// NewNeo4jService creates a new Neo4jService instance
func NewNeo4jService(driver neo4j.Driver, database string, transportMode string, neo4jMCPVersion string) (*Neo4jService, error) {
	if driver == nil {
		return nil, fmt.Errorf("driver cannot be nil")
	}

	return &Neo4jService{
		driver:          driver,
		database:        database,
		transportMode:   transportMode,
		neo4jMCPVersion: neo4jMCPVersion,
	}, nil
}

// buildQueryOptions builds Neo4j query options based on transport mode.
// For HTTP mode: extracts credentials from context and uses impersonation.
// Supports both Bearer token auth (preferred for SSO/OAuth) and Basic Auth (fallback).
// Bearer tokens are passed directly to Neo4j for SSO/OAuth scenarios.
// If credentials are absent, they are not added to the query options (driver defaults apply).
// For STDIO mode: uses driver's built-in credentials (no auth token added).
// The baseOptions parameter allows adding routing-specific options (readers/writers).
// TxMetadata is added to recognize queries coming from Neo4j MCP.
func (s *Neo4jService) buildQueryOptions(ctx context.Context, baseOptions ...neo4j.ExecuteQueryConfigurationOption) []neo4j.ExecuteQueryConfigurationOption {
	txMetadata := neo4j.WithTxMetadata(map[string]any{"app": strings.Join([]string{appName, s.neo4jMCPVersion}, "/")})

	queryOptions := []neo4j.ExecuteQueryConfigurationOption{
		neo4j.ExecuteQueryWithDatabase(s.database),
		neo4j.ExecuteQueryWithTransactionConfig(txMetadata),
	}

	// Add any base options (routing, etc.)
	queryOptions = append(queryOptions, baseOptions...)

	// For HTTP mode, extract credentials from context and use impersonation
	if s.transportMode == config.TransportModeHTTP {
		authToken := s.getHTTPAuthToken(ctx)
		if authToken != nil {
			queryOptions = append(queryOptions, neo4j.ExecuteQueryWithAuthToken(*authToken))
		}
	}
	// For STDIO mode, driver's built-in credentials are used automatically (no auth token needed)
	return queryOptions
}

// VerifyConnectivity checks the driver can establish a valid connection with a Neo4j instance;
func (s *Neo4jService) VerifyConnectivity(ctx context.Context) error {
	// For HTTP mode, extract credentials from context
	if s.transportMode == config.TransportModeHTTP {
		authToken := s.getHTTPAuthToken(ctx)
		if authToken != nil {
			if err := s.driver.VerifyAuthentication(ctx, authToken); err != nil {
				slog.Error("Failed to verify database connectivity", "error", err.Error())
				return err
			}
			return nil
		}
	}
	// Verify database connectivity
	if err := s.driver.VerifyConnectivity(ctx); err != nil {
		slog.Error("Failed to verify database connectivity", "error", err.Error())
		return err
	}
	return nil
}

// Collect HTTP Auth token from Context.
func (s *Neo4jService) getHTTPAuthToken(ctx context.Context) *neo4j.AuthToken {
	if token, hasBearerToken := auth.GetBearerToken(ctx); hasBearerToken {
		authToken := neo4j.BearerAuth(token)
		return &authToken
	} else if username, password, hasBasicAuth := auth.GetBasicAuthCredentials(ctx); hasBasicAuth {
		// Fall back to basic auth
		authToken := neo4j.BasicAuth(username, password, "")
		return &authToken
	}
	return nil
}

// ExecuteReadQuery executes a read-only Cypher query and returns raw records
func (s *Neo4jService) ExecuteReadQuery(ctx context.Context, cypher string, params map[string]any) ([]*neo4j.Record, error) {
	queryOptions := s.buildQueryOptions(ctx, neo4j.ExecuteQueryWithReadersRouting())

	res, err := neo4j.ExecuteQuery(ctx, s.driver, cypher, params, neo4j.EagerResultTransformer, queryOptions...)
	if err != nil {
		wrappedErr := fmt.Errorf("failed to execute read query: %w", err)
		slog.Error("Error in ExecuteReadQuery", "error", wrappedErr)

		return nil, wrappedErr
	}

	return res.Records, nil
}

// ExecuteWriteQuery executes a write-only Cypher query and returns raw records
func (s *Neo4jService) ExecuteWriteQuery(ctx context.Context, cypher string, params map[string]any) ([]*neo4j.Record, error) {
	queryOptions := s.buildQueryOptions(ctx, neo4j.ExecuteQueryWithWritersRouting())

	res, err := neo4j.ExecuteQuery(ctx, s.driver, cypher, params, neo4j.EagerResultTransformer, queryOptions...)
	if err != nil {
		wrappedErr := fmt.Errorf("failed to execute write query: %w", err)
		slog.Error("Error in ExecuteWriteQuery", "error", wrappedErr)
		return nil, wrappedErr
	}

	return res.Records, nil
}

// GetQueryType prefixes the provided query with EXPLAIN and returns the query type (e.g. 'r' for read, 'w' for write, 'rw' etc.)
// This allows read-only tools to determine if a query is safe to run in read-only context.
func (s *Neo4jService) GetQueryType(ctx context.Context, cypher string, params map[string]any) (neo4j.QueryType, error) {
	explainedQuery := strings.Join([]string{"EXPLAIN", cypher}, " ")

	queryOptions := s.buildQueryOptions(ctx)

	res, err := neo4j.ExecuteQuery(ctx, s.driver, explainedQuery, params, neo4j.EagerResultTransformer, queryOptions...)
	if err != nil {
		wrappedErr := fmt.Errorf("error during GetQueryType: %w", err)
		slog.Error("Error during GetQueryType", "error", wrappedErr)
		return neo4j.QueryTypeUnknown, wrappedErr
	}

	if res.Summary == nil {
		err := fmt.Errorf("error during GetQueryType: no summary returned for explained query")
		slog.Error("Error during GetQueryType", "error", err)
		return neo4j.QueryTypeUnknown, err
	}

	return res.Summary.QueryType(), nil

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
