package tools

import (
	"github.com/mark3labs/mcp-go/mcp"
)

func ListOfGDSFunctionsSpec() mcp.Tool {
	return mcp.NewTool("list-graph-data-functions",
		mcp.WithDescription("Use this tool to discover what graph science and analytics functions are available in the current Neo4j environment. It returns a structured list describing each function — what it does, how to use it, the inputs it needs, and what kind of results it produces.\n\nCall this tool whenever you need to perform graph analysis or reasoning tasks (for example: measuring node importance, finding communities, or comparing similarity between nodes) but don’t yet know which specific functions are available or how to call them.\n\nThe tool helps you understand the analytical capabilities of the system so that you can plan or compose the right graph science operations automatically. An empty response indicates that GDS is not installed and the user should be told to install it. Remember to use unique names for graph data science projections to avoid collisions and to drop them afterwards to save memory."),
		mcp.WithTitleAnnotation("Lists available graph science functions and explains what each does, so you know which analytical tools you can use and how to use them."),
		mcp.WithReadOnlyHintAnnotation(false),
		mcp.WithDestructiveHintAnnotation(true),
		mcp.WithIdempotentHintAnnotation(false),
		mcp.WithOpenWorldHintAnnotation(true),
	)
}
