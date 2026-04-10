// Copyright (c) "Neo4j"
// Neo4j Sweden AB [http://neo4j.com]

package gds

import "github.com/mark3labs/mcp-go/mcp"

type CreateGdsProjectionInput struct {
	Name               string `json:"name" jsonschema:"default=mcp-projection,description=The name of the GDS projection to create. Use unique names to avoid collisions and drop projections when they are no longer needed."`
	ProjectionQuery    string `json:"projection_query" jsonschema:"default=MATCH (source) OPTIONAL MATCH (source)-->(target),description=The Cypher query to used to find source and target nodes for the projection. Cannot contain a RETURN clause"`
	SourceNodeVariable string `json:"source_node_variable" jsonschema:"default=source,description=The variable name in the projection query that identifies the source nodes."`
	TargetNodeVariable string `json:"target_node_variable" jsonschema:"default=target,description=The variable name in the projection query that identifies the target nodes."`
	// TODO: add support for data config
	// TODO: add support for config parameters, such as concurrency, writeConcurrency, etc...
}

func CreateGdsProjectionSpec() mcp.Tool {
	return mcp.NewTool("create-gds-projection",
		mcp.WithDescription("Use this tool to create a graph data science projection in Neo4j"),
		mcp.WithTitleAnnotation("Create a GDS projection in Neo4j"),
		mcp.WithReadOnlyHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(true),
		mcp.WithOpenWorldHintAnnotation(true),
		mcp.WithInputSchema[CreateGdsProjectionInput](),
	)
}
