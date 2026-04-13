// Copyright (c) "Neo4j"
// Neo4j Sweden AB [http://neo4j.com]

package gds

import "github.com/mark3labs/mcp-go/mcp"

type DropGdsProjectionInput struct {
	ProjectionName string `json:"projection_name" jsonschema:"default=mcp-projection,description=The name of the GDS projection to drop. This should match the name of the projection created using the create-gds-projection tool."`
}

func DropGdsProjectionSpec() mcp.Tool {
	return mcp.NewTool("drop-gds-projection",
		mcp.WithDescription("Use this tool to drop a gds projection in Neo4j when it is no longer needed to save resources"),
		mcp.WithTitleAnnotation("Drop a GDS projection in Neo4j"),
		mcp.WithReadOnlyHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(true),
		mcp.WithOpenWorldHintAnnotation(true),
		mcp.WithInputSchema[DropGdsProjectionInput](),
	)
}
