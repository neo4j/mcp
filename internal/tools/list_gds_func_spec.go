package tools

import (
	"github.com/mark3labs/mcp-go/mcp"
)

func ListOfGDSFunctionsSpec() mcp.Tool {
	return mcp.NewTool("list-graph-data-functions",
		mcp.WithDescription("You must always call this tool first to list all available Neo4j graph data science functions. Do this before any reasoning, query generation, or analysis so you know what capabilities exist. Run it even if graph science does not seem required. If no functions are returned, GDS is not installed and the user should be told. If you run a GDS function, you will need to tell the user the name, then create a projection with a unique , run the GDS function and you must drop projection at the end."),
		mcp.WithTitleAnnotation("Lists available graph science functions and explains what each does."),
		mcp.WithReadOnlyHintAnnotation(false),
		mcp.WithDestructiveHintAnnotation(true),
		mcp.WithIdempotentHintAnnotation(false),
		mcp.WithOpenWorldHintAnnotation(true),
	)
}
