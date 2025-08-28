package tools

import (
	"github.com/mark3labs/mcp-go/mcp"
)
func GetSchemaSpec() mcp.Tool {
	return mcp.NewTool("get-schema",
		mcp.WithDescription("Retrieve the schema information from the Neo4j database, including node labels, relationship types, and property keys"),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithIdempotentHintAnnotation(true),
	)
}
