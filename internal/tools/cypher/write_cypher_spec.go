// Copyright (c) "Neo4j"
// Neo4j Sweden AB [http://neo4j.com]

package cypher

import (
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/neo4j/mcp/internal/tools"
)

type WriteCypherInput struct {
	Query  string `json:"query" jsonschema:"The Cypher query to execute"`
	Params Params `json:"params,omitempty" jsonschema:"Parameters to pass to the Cypher query"`
}

func WriteCypherSpec() *mcp.Tool {
	return &mcp.Tool{
		Name:        "write-cypher",
		Description: "write-cypher executes any arbitrary Cypher query, with write access, against the user-configured Neo4j database.",
		Annotations: &mcp.ToolAnnotations{
			Title:           "Write Cypher",
			ReadOnlyHint:    false,
			DestructiveHint: tools.BoolPtr(true),
			IdempotentHint:  false,
			OpenWorldHint:   tools.BoolPtr(true),
		},
	}
}
