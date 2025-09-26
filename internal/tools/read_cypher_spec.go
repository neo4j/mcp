package tools

import (
	"github.com/mark3labs/mcp-go/mcp"
)

type ReadCypherInput struct {
	Query  string         `json:"query" jsonschema:"default=MATCH(n) RETURN n,description=The Cypher query to execute"`
	Params map[string]any `json:"params" jsonschema:"description=Parameters to pass to the Cypher query"`
}

func ReadCypherSpec() mcp.Tool {
	return mcp.NewTool("read-cypher",
		mcp.WithDescription("read-cypher executes any arbitrary readonly Cypher query against the user-configured Neo4j database."),
		mcp.WithInputSchema[ReadCypherInput](),
		mcp.WithTitleAnnotation("Read Cypher"),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
		mcp.WithOpenWorldHintAnnotation(true),
	)
}
