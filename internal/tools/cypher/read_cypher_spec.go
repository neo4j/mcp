// Copyright (c) "Neo4j"
// Neo4j Sweden AB [http://neo4j.com]

package cypher

import (
	"github.com/mark3labs/mcp-go/mcp"
)

type ReadCypherInput struct {
	Query  string `json:"query" jsonschema:"default=MATCH(n) RETURN n,description=The Cypher query to execute"`
	Params Params `json:"params,omitempty" jsonschema:"default={},description=Parameters to pass to the Cypher query"`
}

func ReadCypherSpec() mcp.Tool {
	return mcp.NewTool("read-cypher",
		mcp.WithDescription("read-cypher can run only read-only Cypher statements. For write operations (CREATE, MERGE, DELETE, SET, etc...), schema/admin commands, or PROFILE queries, use write-cypher instead."),
		mcp.WithInputSchema[ReadCypherInput](),
		mcp.WithTitleAnnotation("Read Cypher"),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
		mcp.WithOpenWorldHintAnnotation(true),
	)
}
