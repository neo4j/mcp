package cypher

import (
	"github.com/mark3labs/mcp-go/mcp"
)

type WriteCypherInput struct {
	Query  string         `json:"query" jsonschema:"default=MATCH(n) RETURN n,description=The Cypher query to execute"`
	Params map[string]any `json:"params" jsonschema:"default={},description=Parameters to pass to the Cypher query"`
}

// GetParams returns the params map
func (w *WriteCypherInput) GetParams() map[string]any {
	return w.Params
}

// SetParams sets the params map
func (w *WriteCypherInput) SetParams(params map[string]any) {
	w.Params = params
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
