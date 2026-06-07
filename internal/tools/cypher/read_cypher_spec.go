// Copyright (c) "Neo4j"
// Neo4j Sweden AB [http://neo4j.com]

package cypher

import (
	"encoding/json"

	"github.com/mark3labs/mcp-go/mcp"
)

type ReadCypherInput struct {
	Query  string `json:"query"`
	Params Params `json:"params,omitempty"`
}

func ReadCypherSpec() mcp.Tool {
	return mcp.NewTool("read-cypher",
		mcp.WithDescription("read-cypher can run only read-only Cypher statements. For write operations (CREATE, MERGE, DELETE, SET, etc...), schema/admin commands, or PROFILE queries, use write-cypher instead."),
		rawInputSchema(json.RawMessage(`{
			"type": "object",
			"properties": {
				"query": {
					"type": "string",
					"description": "The Cypher query to execute"
				},
				"params": {
					"type": "object",
					"description": "Parameters to pass to the Cypher query",
					"additionalProperties": true
				}
			},
			"required": ["query"],
			"additionalProperties": false
		}`)),
		mcp.WithTitleAnnotation("Read Cypher"),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
		mcp.WithOpenWorldHintAnnotation(true),
	)
}
