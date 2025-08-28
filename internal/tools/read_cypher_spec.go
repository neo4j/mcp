package tools

import (
	"github.com/mark3labs/mcp-go/mcp"
)

// ReadCypherInput defines the input schema for the read-cypher tool
type ReadCypherInput struct {
	Query  string         `json:"query" jsonschema:"default=MATCH(n) RETURN n,description=The Cypher query to execute"`
	Params map[string]any `json:"params" jsonschema:"description=Parameters to pass to the Cypher query"`
}

// ReadCypherSpec returns the MCP tool specification for read-cypher
func ReadCypherSpec() mcp.Tool {
	return mcp.NewTool("read-cypher",
		mcp.WithDescription("Perform a read-only Cypher against a Neo4j database"),
		mcp.WithInputSchema[ReadCypherInput](),
		mcp.WithReadOnlyHintAnnotation(true),
	)
}
