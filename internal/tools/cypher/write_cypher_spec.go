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
	// OmitEmbeddings is opt-in so existing clients see no change in results.
	OmitEmbeddings bool `json:"omitEmbeddings,omitempty" jsonschema:"When true, replaces embedding-like values (native VECTOR values and lists of 128 or more floats) with a short placeholder so they do not fill the context window. Defaults to false."`
}

func WriteCypherSpec() *mcp.Tool {
	return &mcp.Tool{
		Name:        "write-cypher",
		Description: "write-cypher executes any arbitrary Cypher query, with write access, against the user-configured Neo4j database. Prefer returning only needed properties over whole nodes to avoid context bloat from large responses.",
		Annotations: &mcp.ToolAnnotations{
			Title:           "Write Cypher",
			ReadOnlyHint:    false,
			DestructiveHint: tools.BoolPtr(true),
			IdempotentHint:  false,
			OpenWorldHint:   tools.BoolPtr(true),
		},
	}
}
