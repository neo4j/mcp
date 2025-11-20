package cypher

import (
	"github.com/mark3labs/mcp-go/mcp"
)

type WriteCypherInput struct {
	Query  string       `json:"query" jsonschema:"default=MATCH(n) RETURN n,description=The Cypher query to execute"`
	Params CypherParams `json:"params" jsonschema:"default={},description=Parameters to pass to the Cypher query"`
}

func WriteCypherSpec() mcp.Tool {
	return mcp.NewTool("write-cypher",
		mcp.WithDescription("write-cypher executes any arbitrary Cypher query, with write access, against the user-configured Neo4j database."),
		mcp.WithInputSchema[WriteCypherInput](),
		mcp.WithTitleAnnotation("Write Cypher"),
		mcp.WithReadOnlyHintAnnotation(false),
		mcp.WithDestructiveHintAnnotation(true),
		mcp.WithIdempotentHintAnnotation(false),
		mcp.WithOpenWorldHintAnnotation(true),
	)
}
