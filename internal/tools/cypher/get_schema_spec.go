package cypher

import (
	"github.com/mark3labs/mcp-go/mcp"
)

type GetSchemaInput struct {
	SampleSize int64 `json:"sample-size" jsonschema:"default=100,description=The number of nodes to sample to infer the database schema."`
}

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
		mcp.WithInputSchema[GetSchemaInput](),
	)
}
