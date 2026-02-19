//
// Copyright (c) "Neo4j"
// Neo4j Sweden AB [http://neo4j.com]

package gds

import "github.com/mark3labs/mcp-go/mcp"

func ListGDSProceduresSpec() mcp.Tool {
	return mcp.NewTool("list-gds-procedures",
		mcp.WithDescription(
			"Use this tool to discover what graph science and analytics functions are available in the current Neo4j environment. "+
				"It returns a structured list describing each function — what it does, how to use it, the inputs it needs, and what kind of results it produces. "+
				"Do this before any reasoning, query generation, or analysis so you know what capabilities exist. "+
				"Graph science and analytics functions help you with centrality, community detection, similarity, path finding, and identifying dependencies between nodes. "+
				"The tool helps you understand the analytical capabilities of the system so that you can plan or compose the right graph science operations automatically. "+
				"An empty response indicates that GDS is not installed and the user should be told to install it. "+
				"Remember to use unique names for graph data science projections to avoid collisions and to drop them afterwards to save memory. "+
				"You must always tell the user the function you will use.",
		),
		mcp.WithTitleAnnotation("List available Neo4j GDS procedures"),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithIdempotentHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithOpenWorldHintAnnotation(true),
	)
}
