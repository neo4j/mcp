package tools

import (
	"github.com/neo4j/mcp/internal/database"
	"github.com/neo4j/mcp/internal/logger"
)

// ToolDependencies contains all dependencies needed by tools
type ToolDependencies struct {
	DBService database.Service
	Log       *logger.Service
}
