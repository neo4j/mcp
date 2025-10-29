package tools

import (
	"github.com/neo4j/mcp/internal/database"
)

// ToolDependencies contains all dependencies needed by tools
type ToolDependencies struct {
	DBService database.Service
}
