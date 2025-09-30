package database

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

// Neo4jService is the concrete implementation of DatabaseService
type Neo4jService struct {
	Driver Driver
}

// NewNeo4jService creates a new Neo4jService instance
func NewNeo4jService(driver neo4j.DriverWithContext) DatabaseService {
	return &Neo4jService{
		Driver: &neo4jDriverAdapter{driver: driver},
	}
}

// NewNeo4jServiceWithDriver creates a Neo4jService with a custom Driver for testing
func NewNeo4jServiceWithDriver(driver Driver) DatabaseService {
	return &Neo4jService{
		Driver: driver,
	}
}

// ExecuteReadQuery executes a read-only Cypher query and returns raw records
func (s *Neo4jService) ExecuteReadQuery(ctx context.Context, cypher string, params map[string]any, database string) ([]*neo4j.Record, error) {
	if s.Driver == nil {
		err := fmt.Errorf("driver is not initialized")
		log.Printf("Error in ExecuteReadQuery: %v", err)
		return nil, err
	}

	session, err := s.Driver.NewSession(ctx, database)
	if err != nil {
		wrappedErr := fmt.Errorf("failed to create session: %w", err)
		log.Printf("Error in ExecuteReadQuery: %v", wrappedErr)
		return nil, wrappedErr
	}
	defer func() {
		if closeErr := session.Close(ctx); closeErr != nil {
			log.Printf("Error closing session: %v", closeErr)
		}
	}()

	result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		res, err := tx.Run(ctx, cypher, params)
		if err != nil {
			return nil, err
		}
		return res.Collect(ctx)
	})

	if err != nil {
		wrappedErr := fmt.Errorf("failed to execute read query: %w", err)
		log.Printf("Error in ExecuteReadQuery: %v", wrappedErr)
		return nil, wrappedErr
	}

	records, ok := result.([]*neo4j.Record)
	if !ok {
		err := fmt.Errorf("unexpected result type from read transaction: expected []*neo4j.Record, got %T", result)
		log.Printf("Error in ExecuteReadQuery: %v", err)
		return nil, err
	}

	return records, nil
}

// ExecuteWriteQuery executes a write-only Cypher query and returns raw records
func (s *Neo4jService) ExecuteWriteQuery(ctx context.Context, cypher string, params map[string]any, database string) ([]*neo4j.Record, error) {
	if s.Driver == nil {
		err := fmt.Errorf("driver is not initialized")
		log.Printf("Error in ExecuteWriteQuery: %v", err)
		return nil, err
	}

	session, err := s.Driver.NewSession(ctx, database)
	if err != nil {
		wrappedErr := fmt.Errorf("failed to create session: %w", err)
		log.Printf("Error in ExecuteWriteQuery: %v", wrappedErr)
		return nil, wrappedErr
	}
	defer func() {
		if closeErr := session.Close(ctx); closeErr != nil {
			log.Printf("Error closing session: %v", closeErr)
		}
	}()

	result, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		res, err := tx.Run(ctx, cypher, params)
		if err != nil {
			return nil, err
		}
		return res.Collect(ctx)
	})

	if err != nil {
		wrappedErr := fmt.Errorf("failed to execute write query: %w", err)
		log.Printf("Error in ExecuteWriteQuery: %v", wrappedErr)
		return nil, wrappedErr
	}

	records, ok := result.([]*neo4j.Record)
	if !ok {
		err := fmt.Errorf("unexpected result type from write transaction: expected []*neo4j.Record, got %T", result)
		log.Printf("Error in ExecuteWriteQuery: %v", err)
		return nil, err
	}

	return records, nil
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

	return string(formattedResponse), nil
}
