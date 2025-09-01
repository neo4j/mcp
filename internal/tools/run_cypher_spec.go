package tools

import (
	"github.com/mark3labs/mcp-go/mcp"
)

type RunCypherInput struct {
	Query  string         `json:"query" jsonschema:"default=MATCH(n) RETURN n,description=The Cypher query to execute"`
	Params map[string]any `json:"params" jsonschema:"description=Parameters to pass to the Cypher query"`
}

func RunCypherSpec() mcp.Tool {
	return mcp.NewTool("run-cypher",
		mcp.WithDescription("Run a Cypher against a Neo4j database"),
		mcp.WithInputSchema[RunCypherInput](),
		mcp.WithReadOnlyHintAnnotation(false),
		mcp.WithDestructiveHintAnnotation(true),
	)
}
