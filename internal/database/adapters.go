package database

import (
	"context"
	"fmt"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

// neo4jDriverAdapter wraps neo4j.DriverWithContext to implement our Driver interface
type neo4jDriverAdapter struct {
	driver neo4j.DriverWithContext
}

// NewSession creates a new session for the specified database
func (a *neo4jDriverAdapter) NewSession(ctx context.Context, database string) (Session, error) {
	if a.driver == nil {
		return nil, fmt.Errorf("Neo4j driver is not initialized")
	}
	session := a.driver.NewSession(ctx, neo4j.SessionConfig{
		DatabaseName: database,
	})
	return &neo4jSessionAdapter{session: session}, nil
}

// neo4jSessionAdapter wraps neo4j.SessionWithContext to implement our Session interface
type neo4jSessionAdapter struct {
	session neo4j.SessionWithContext
}

// ExecuteRead executes a read transaction
func (a *neo4jSessionAdapter) ExecuteRead(ctx context.Context, work func(tx neo4j.ManagedTransaction) (any, error)) (any, error) {
	return a.session.ExecuteRead(ctx, work)
}

// ExecuteWrite executes a write transaction
func (a *neo4jSessionAdapter) ExecuteWrite(ctx context.Context, work func(tx neo4j.ManagedTransaction) (any, error)) (any, error) {
	return a.session.ExecuteWrite(ctx, work)
}

// Close closes the session
func (a *neo4jSessionAdapter) Close(ctx context.Context) error {
	return a.session.Close(ctx)
}