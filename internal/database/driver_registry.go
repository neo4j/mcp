// Copyright (c) "Neo4j"
// Neo4j Sweden AB [http://neo4j.com]

package database

import (
	"fmt"

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
	driver, err := neo4j.NewDriver(boltURI, neo4j.NoAuth())
	if err != nil {
		return nil, fmt.Errorf("failed to create Neo4j driver for %s: %w", boltURI, err)
	}
	return driver, nil
}
