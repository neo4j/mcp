package tools

import (
	"github.com/mark3labs/mcp-go/mcp"
)

type GetGDSFunctionDetailsInput struct {
	FunctionName string `json:"functionName" jsonschema:"description=Gets the details of the GDS function with this name."`
}

func GetGDSFunctionDetailsSpec() mcp.Tool {
	return mcp.NewTool("get-graph-data-functions",
		mcp.WithDescription("Use this tool to obtain the details of GDS function"),
		mcp.WithTitleAnnotation("Gets the detalis for a GDS function"),
		mcp.WithInputSchema[GetGDSFunctionDetailsInput](),
		mcp.WithReadOnlyHintAnnotation(false),
		mcp.WithDestructiveHintAnnotation(true),
		mcp.WithIdempotentHintAnnotation(false),
		mcp.WithOpenWorldHintAnnotation(true),
	)
}
