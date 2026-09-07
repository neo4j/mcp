// Copyright (c) "Neo4j"
// Neo4j Sweden AB [http://neo4j.com]

package tools

import (
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// BoolPtr returns a pointer to b, for populating the *bool fields of
// go-sdk's mcp.ToolAnnotations (DestructiveHint, OpenWorldHint).
func BoolPtr(b bool) *bool {
	return &b
}

// NewToolTextResult builds a successful tool result from plain text.
func NewToolTextResult(text string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: text},
		},
	}
}

// NewToolErrorResult builds a tool-level error result (IsError: true), not a protocol-level error,
// so the calling LLM can see the message and self-correct.
func NewToolErrorResult(message string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: message},
		},
		IsError: true,
	}
}
