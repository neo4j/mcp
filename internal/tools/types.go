// Copyright (c) "Neo4j"
// Neo4j Sweden AB [http://neo4j.com]

package tools

import (
	"github.com/neo4j/mcp/internal/analytics"
	"github.com/neo4j/mcp/internal/database"
)

// ToolDependencies contains all dependencies needed by tools
type ToolDependencies struct {
	DBService        database.Service
	AnalyticsService analytics.Service
	SchemaSampleSize int
}
