package database

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

// Neo4jSessionFactory is the concrete implementation of SessionFactory
type Neo4jSessionFactory struct {
	driver neo4j.DriverWithContext
}

// NewSession creates a new Neo4j session for the specified database
func (f *Neo4jSessionFactory) NewSession(ctx context.Context, database string) (neo4j.SessionWithContext, error) {
	if f.driver == nil {
		return nil, fmt.Errorf("error in NewSession: Neo4j driver is not initialized")
	}
	return f.driver.NewSession(ctx, neo4j.SessionConfig{
		DatabaseName: database,
	}), nil
}

// Neo4jService is the concrete implementation of DatabaseService
type Neo4jService struct {
	sessionFactory SessionFactory
}

// NewNeo4jService creates a new Neo4jService instance
func NewNeo4jService(driver neo4j.DriverWithContext) DatabaseService {
	return &Neo4jService{
		sessionFactory: &Neo4jSessionFactory{driver: driver},
	}
}

// closeSession safely closes a Neo4j session and logs any errors
func (s *Neo4jService) closeSession(ctx context.Context, session neo4j.SessionWithContext) {
	if closeErr := session.Close(ctx); closeErr != nil {
		log.Printf("Error closing session: %v", closeErr)
	}
}

// ExecuteReadQuery executes a read-only Cypher query and returns raw records
func (s *Neo4jService) ExecuteReadQuery(ctx context.Context, cypher string, params map[string]any, database string) ([]*neo4j.Record, error) {
	if s.sessionFactory == nil {
		err := fmt.Errorf("Neo4j session factory is not initialized")
		log.Printf("Error in ExecuteReadQuery: %v", err)
		return nil, err
	}
	session, sessionErr := s.sessionFactory.NewSession(ctx, database)
	if sessionErr != nil {
		log.Printf("Error in ExecuteReadQuery: %v", sessionErr)
		return nil, sessionErr
	}
	if session == nil {
		err := fmt.Errorf("session factory returned nil session")
		log.Printf("Error in ExecuteReadQuery: %v", err)
		return nil, err
	}
	defer s.closeSession(ctx, session)

	result, err := session.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		res, err := tx.Run(ctx, cypher, params)
		if err != nil {
			return nil, err
		}
		records, collectErr := res.Collect(ctx)
		if collectErr != nil {
			return nil, collectErr
		}
		return records, nil
	})

	if err != nil {
		wrappedErr := fmt.Errorf("failed to execute read query: %w", err)
		log.Printf("Error in ExecuteReadQuery: %v", wrappedErr)
		return nil, wrappedErr
	}

	// Safe type assertion - we know the transaction function returns []*neo4j.Record
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
	if s.sessionFactory == nil {
		err := fmt.Errorf("Neo4j session factory is not initialized")
		log.Printf("Error in ExecuteWriteQuery: %v", err)
		return nil, err
	}
	session, sessionErr := s.sessionFactory.NewSession(ctx, database)
	if sessionErr != nil {
		log.Printf("Error in ExecuteWriteQuery: %v", sessionErr)
		return nil, sessionErr
	}
	if session == nil {
		err := fmt.Errorf("session factory returned nil session")
		log.Printf("Error in ExecuteWriteQuery: %v", err)
		return nil, err
	}
	defer s.closeSession(ctx, session)

	result, err := session.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		res, err := tx.Run(ctx, cypher, params)
		if err != nil {
			return nil, err
		}
		records, collectErr := res.Collect(ctx)
		if collectErr != nil {
			return nil, collectErr
		}
		return records, nil
	})

	if err != nil {
		wrappedErr := fmt.Errorf("failed to execute write query: %w", err)
		log.Printf("Error in ExecuteWriteQuery: %v", wrappedErr)
		return nil, wrappedErr
	}

	// Safe type assertion - we know the transaction function returns []*neo4j.Record
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

	formattedResponseStr := string(formattedResponse)

	return formattedResponseStr, nil
}
