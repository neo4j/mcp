package server

import (
	"context"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
)

func TestWithPermissionCheck(t *testing.T) {
	// Mock handler that returns success
	successHandler := func(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return mcp.NewToolResultText("Tool executed successfully"), nil
	}

	tests := []struct {
		name           string
		userContext    *UserContext
		toolName       string
		expectError    bool
		expectedOutput string
	}{
		{
			name:           "auth disabled - allows execution",
			userContext:    nil,
			toolName:       "get-schema",
			expectError:    false,
			expectedOutput: "Tool executed successfully",
		},
		{
			name: "user has required permission - allows execution",
			userContext: &UserContext{
				UserID:      "user123",
				Permissions: []string{"read:schema"},
			},
			toolName:       "get-schema",
			expectError:    false,
			expectedOutput: "Tool executed successfully",
		},
		{
			name: "user has admin:all - allows execution",
			userContext: &UserContext{
				UserID:      "admin123",
				Permissions: []string{"admin:all"},
			},
			toolName:       "write-cypher",
			expectError:    false,
			expectedOutput: "Tool executed successfully",
		},
		{
			name: "user lacks permission - denies execution",
			userContext: &UserContext{
				UserID:      "user456",
				Permissions: []string{"read:schema"},
			},
			toolName:       "write-cypher",
			expectError:    true,
			expectedOutput: "insufficient permissions",
		},
		{
			name: "unknown tool - denies execution",
			userContext: &UserContext{
				UserID:      "user789",
				Permissions: []string{"admin:all"},
			},
			toolName:       "unknown-tool",
			expectError:    true,
			expectedOutput: "not configured for permission checking",
		},
		{
			name: "user has read:data - allows read-cypher",
			userContext: &UserContext{
				UserID:      "reader123",
				Permissions: []string{"read:data"},
			},
			toolName:       "read-cypher",
			expectError:    false,
			expectedOutput: "Tool executed successfully",
		},
		{
			name: "user has write:data - allows write-cypher",
			userContext: &UserContext{
				UserID:      "writer123",
				Permissions: []string{"write:data"},
			},
			toolName:       "write-cypher",
			expectError:    false,
			expectedOutput: "Tool executed successfully",
		},
		{
			name: "user has read:gds - allows list-gds-procedures",
			userContext: &UserContext{
				UserID:      "gds123",
				Permissions: []string{"read:gds"},
			},
			toolName:       "list-gds-procedures",
			expectError:    false,
			expectedOutput: "Tool executed successfully",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup context
			ctx := context.Background()
			if tt.userContext != nil {
				ctx = context.WithValue(ctx, contextKeyUserCtx, tt.userContext)
			}

			// Create request with tool name
			request := mcp.CallToolRequest{
				Params: mcp.CallToolParams{
					Name:      tt.toolName,
					Arguments: map[string]interface{}{},
				},
			}

			// Wrap handler with permission check
			wrappedHandler := WithPermissionCheck(successHandler)

			// Execute wrapped handler
			result, err := wrappedHandler(ctx, request)

			// Verify no Go errors are returned (MCP errors are in result)
			if err != nil {
				t.Errorf("Unexpected Go error: %v", err)
			}

			// Check result
			if result == nil {
				t.Fatal("Result should not be nil")
			}

			// Extract text content from result
			if len(result.Content) == 0 {
				t.Fatal("Result should have content")
			}

			textContent, ok := result.Content[0].(mcp.TextContent)
			if !ok {
				t.Fatal("Result content should be TextContent")
			}

			// Verify expectations
			if tt.expectError {
				// Should have IsError set to true
				if !result.IsError {
					t.Error("Expected IsError to be true")
				}

				// Check error message contains expected text
				if !strings.Contains(textContent.Text, tt.expectedOutput) {
					t.Errorf("Expected error message to contain '%s', got '%s'", tt.expectedOutput, textContent.Text)
				}
			} else {
				// Should not have IsError set
				if result.IsError {
					t.Errorf("Expected no error, but IsError is true. Content: %s", textContent.Text)
				}

				// Check success message
				if !strings.Contains(textContent.Text, tt.expectedOutput) {
					t.Errorf("Expected output to contain '%s', got '%s'", tt.expectedOutput, textContent.Text)
				}
			}
		})
	}
}

func TestWithPermissionCheck_HandlerError(t *testing.T) {
	// Mock handler that returns an error result
	errorHandler := func(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return mcp.NewToolResultError("Handler internal error"), nil
	}

	// Setup context with valid permissions
	ctx := context.Background()
	userCtx := &UserContext{
		UserID:      "user123",
		Permissions: []string{"read:schema"},
	}
	ctx = context.WithValue(ctx, contextKeyUserCtx, userCtx)

	// Create request
	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "get-schema",
			Arguments: map[string]interface{}{},
		},
	}

	// Wrap handler
	wrappedHandler := WithPermissionCheck(errorHandler)

	// Execute
	result, err := wrappedHandler(ctx, request)

	// Verify no Go errors
	if err != nil {
		t.Errorf("Unexpected Go error: %v", err)
	}

	// Verify handler error is preserved
	if result == nil {
		t.Fatal("Result should not be nil")
	}

	if !result.IsError {
		t.Error("Expected IsError to be true from handler")
	}

	textContent, ok := result.Content[0].(mcp.TextContent)
	if !ok {
		t.Fatal("Result content should be TextContent")
	}
	if !strings.Contains(textContent.Text, "Handler internal error") {
		t.Errorf("Expected handler error to be preserved, got: %s", textContent.Text)
	}
}
