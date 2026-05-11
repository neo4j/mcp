// Copyright (c) "Neo4j"
// Neo4j Sweden AB [http://neo4j.com]

package cypher

import (
	"github.com/mark3labs/mcp-go/mcp"
)

type ReadCypherInput struct {
	Query  string `json:"query" jsonschema:"The Cypher query to execute"`
	Params Params `json:"params,omitempty" jsonschema:"Parameters to pass to the Cypher query"`
}

func ReadCypherSpec() mcp.Tool {
	return mcp.NewTool("read-cypher",
		mcp.WithDescription("read-cypher executes read-only Cypher queries that do not modify database data. Before execution, it validates queries using EXPLAIN and Neo4j's query classification — only queries classified as read-only ('r') are permitted. Note: custom procedures or functions incorrectly classified as read-only by Neo4j may bypass this check; ensuring correct classification is the responsibility of the procedure/function maintainer. For write operations, schema/admin commands, or PROFILE queries, use write-cypher instead."),
		mcp.WithInputSchema[ReadCypherInput](),
		mcp.WithTitleAnnotation("Read Cypher"),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
		mcp.WithOpenWorldHintAnnotation(true),
	)
}
