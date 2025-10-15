package tools

import (
	"github.com/mark3labs/mcp-go/mcp"
)

// ExecuteGDSFunctionInput defines the input parameters for the execute-gds-function tool
type ExecuteGDSFunctionInput struct {
	// FunctionName is the name of the GDS function to execute (e.g., "gds.pageRank.stream")
	FunctionName string `json:"function_name"  jsonschema:"default=MATCH(n) RETURN n,description=FunctionName is the name of the GDS function to execute (e.g., gds.pageRank.stream)"`

	// FunctionParams are the parameters to pass to the GDS function
	// This should be a JSON object with the function's parameters
	FunctionParams map[string]interface{} `json:"function_params"  jsonschema:"default=he parameters to pass to the GDS function. This should be a JSON object"`

	// ProjectionName is the name to use for the GDS projection (optional, will be auto-generated if not provided)
	ProjectionName *string `json:"projection_name,omitempty"  jsonschema:"description=The name of the graph projection you created before calling this tool."`
}

// ExecuteGDSFunctionSpec returns the MCP tool specification for executing GDS functions
func ExecuteGDSFunctionSpec() mcp.Tool {
	return mcp.NewTool("execute-gds-function",
		mcp.WithDescription("After creating a projection, call execute-gds-function with the detailed instructions you got from using get-gds-function-details and include the name of the projection."),
		mcp.WithInputSchema[ExecuteGDSFunctionInput](),
		mcp.WithTitleAnnotation("Execute GDS Function"),
		mcp.WithReadOnlyHintAnnotation(false),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(false),
		mcp.WithOpenWorldHintAnnotation(true),
	)
}
