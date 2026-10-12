// Copyright (c) "Neo4j"
// Neo4j Sweden AB [http://neo4j.com]

package cypher

import (
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/neo4j/mcp/internal/tools"
)

type ReadCypherInput struct {
	Query  string `json:"query" jsonschema:"The Cypher query to execute"`
	Params Params `json:"params,omitempty" jsonschema:"Parameters to pass to the Cypher query"`
}

func ReadCypherSpec() *mcp.Tool {
	return &mcp.Tool{
		Name:        "read-cypher",
		Description: "read-cypher can run only read-only Cypher statements. For write operations (CREATE, MERGE, DELETE, SET, etc...), schema/admin commands, or PROFILE queries, use write-cypher instead.",
		Annotations: &mcp.ToolAnnotations{
			Title:           "Read Cypher",
			ReadOnlyHint:    true,
			DestructiveHint: tools.BoolPtr(false),
			IdempotentHint:  true,
			OpenWorldHint:   tools.BoolPtr(true),
		},
	}
}
