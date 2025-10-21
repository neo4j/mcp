package server

import (
	"context"
	"log"

	"github.com/mark3labs/mcp-go/mcp"
)

// WithPermissionCheck wraps a tool handler with permission checking
// It extracts the tool name from the request and verifies the user has permission
// before executing the underlying handler
func WithPermissionCheck(handler func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error)) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		// Extract tool name from request
		toolName := request.Params.Name

		// Check if user has permission to execute this tool
		if err := CheckToolPermission(ctx, toolName); err != nil {
			log.Printf("⚠ Permission denied for tool '%s': %v", toolName, err)
			return mcp.NewToolResultError(err.Error()), nil
		}

		// Permission check passed, execute the tool
		return handler(ctx, request)
	}
}
