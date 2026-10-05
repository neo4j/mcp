// Copyright (c) "Neo4j"
// Neo4j Sweden AB [http://neo4j.com]

package gds

import (
	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/neo4j/mcp/internal/tools"
)

func ListGDSProceduresSpec() *mcp.Tool {
	return &mcp.Tool{
		Name: "list-gds-procedures",
		Description: "Use this tool to discover what graph science and analytics functions are available in the current Neo4j environment. " +
			"It returns a structured list describing each function — what it does, how to use it, the inputs it needs, and what kind of results it produces. " +
			"Do this before any reasoning, query generation, or analysis so you know what capabilities exist. " +
			"Graph science and analytics functions help you with centrality, community detection, similarity, path finding, and identifying dependencies between nodes. " +
			"The tool helps you understand the analytical capabilities of the system so that you can plan or compose the right graph science operations automatically. " +
			"An empty response indicates that GDS is not installed and the user should be told to install it. " +
			"Remember to use unique names for graph data science projections to avoid collisions and to drop them afterwards to save memory. " +
			"You must always tell the user the function you will use.",
		Annotations: &mcp.ToolAnnotations{
			Title:           "List available Neo4j GDS procedures",
			ReadOnlyHint:    true,
			DestructiveHint: tools.BoolPtr(false),
			IdempotentHint:  true,
			OpenWorldHint:   tools.BoolPtr(true),
		},
		// Set explicitly rather than left nil otherwise "properties" is omitted entirely 
		// which breaks OpenAI API compatibility (see issue #157).
		InputSchema: &jsonschema.Schema{
			Type:       "object",
			Properties: map[string]*jsonschema.Schema{},
		},
	}
}
