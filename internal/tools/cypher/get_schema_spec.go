package cypher

import (
	"github.com/mark3labs/mcp-go/mcp"
)

// TODO: define an optimized get-schema output and add it as outputSchema
func GetSchemaSpec() mcp.Tool {
	return mcp.NewTool("get-schema",
		mcp.WithDescription(`
		Retrieve the schema information from the Neo4j database, including node labels, relationship types, and property keys.
		If the database contains no data, no schema information is returned.`),
		mcp.WithTitleAnnotation("Get Neo4j Schema"),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithIdempotentHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithOpenWorldHintAnnotation(true),
	)
}
