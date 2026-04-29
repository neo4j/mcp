// Copyright (c) "Neo4j"
// Neo4j Sweden AB [http://neo4j.com]

package database

import (
	"fmt"
	"log/slog"

	"github.com/neo4j/neo4j-go-driver/v6/neo4j"
)

// DriverRegistry creates Neo4j drivers by bolt URI.
type DriverRegistry interface {
	GetDriver(boltURI string) (neo4j.Driver, error)
}

// PerRequestDriverRegistry creates a new driver on every call.
// The caller is responsible for closing the returned driver.
type PerRequestDriverRegistry struct{}

func (r *PerRequestDriverRegistry) GetDriver(boltURI string) (neo4j.Driver, error) {
	// NoAuth here is intentional. Per-request credentials (Basic or Bearer) are
	// applied at query time via neo4j.ExecuteQueryWithAuthToken in buildQueryOptions
	driver, err := neo4j.NewDriver(boltURI, neo4j.NoAuth())
	if err != nil {
		slog.Error("Failed to create Neo4j driver", "boltURI", boltURI, "error", err)
		return nil, fmt.Errorf("failed to create Neo4j driver: %w", err)
	}
	return driver, nil
}
