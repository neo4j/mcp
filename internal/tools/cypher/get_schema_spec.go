// Copyright (c) "Neo4j"
// Neo4j Sweden AB [http://neo4j.com]

package cypher

import (
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/neo4j/mcp/internal/tools"
)

func GetSchemaSpec() *mcp.Tool {
	return &mcp.Tool{
		Name: "get-schema",
		Description: `
		Retrieve the schema information from the Neo4j database, including node labels, relationship types, and property keys.
		If the database contains no data, no schema information is returned.`,
		Annotations: &mcp.ToolAnnotations{
			Title:           "Get Neo4j Schema",
			ReadOnlyHint:    true,
			DestructiveHint: tools.BoolPtr(false),
			IdempotentHint:  true,
			OpenWorldHint:   tools.BoolPtr(true),
		},
	}
}
