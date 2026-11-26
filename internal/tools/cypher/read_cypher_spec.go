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
	// OmitEmbeddings is opt-in so existing clients see no change in results.
	OmitEmbeddings bool `json:"omitEmbeddings,omitempty" jsonschema:"When true, replaces embedding-like values (native VECTOR values and lists of 128 or more floats) with a short placeholder so they do not fill the context window. Defaults to false."`
}

func ReadCypherSpec() *mcp.Tool {
	return &mcp.Tool{
		Name:        "read-cypher",
		Description: "read-cypher can run only read-only Cypher statements. For write operations (CREATE, MERGE, DELETE, SET, etc...), schema/admin commands, or PROFILE queries, use write-cypher instead. Prefer returning only needed properties over whole nodes to avoid context bloat from large responses.",
		Annotations: &mcp.ToolAnnotations{
			Title:           "Read Cypher",
			ReadOnlyHint:    true,
			DestructiveHint: tools.BoolPtr(false),
			IdempotentHint:  true,
			OpenWorldHint:   tools.BoolPtr(true),
		},
	}
}
