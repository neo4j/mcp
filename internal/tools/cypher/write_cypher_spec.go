// Copyright (c) "Neo4j"
// Neo4j Sweden AB [http://neo4j.com]

package cypher

import (
	"encoding/json"

	"github.com/mark3labs/mcp-go/mcp"
)

type WriteCypherInput struct {
	Query  string `json:"query"`
	Params Params `json:"params,omitempty"`
}

func WriteCypherSpec() mcp.Tool {
	return mcp.NewTool("write-cypher",
		mcp.WithDescription("write-cypher executes any arbitrary Cypher query, with write access, against the user-configured Neo4j database."),
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
		mcp.WithTitleAnnotation("Write Cypher"),
		mcp.WithReadOnlyHintAnnotation(false),
		mcp.WithDestructiveHintAnnotation(true),
		mcp.WithIdempotentHintAnnotation(false),
		mcp.WithOpenWorldHintAnnotation(true),
	)
}
