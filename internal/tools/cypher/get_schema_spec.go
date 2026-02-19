//
// Copyright (c) "Neo4j"
// Neo4j Sweden AB [http://neo4j.com]

package cypher

import (
	"github.com/mark3labs/mcp-go/mcp"
)

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
