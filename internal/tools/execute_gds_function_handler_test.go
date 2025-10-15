package tools

import (
	"context"
	"errors"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/neo4j/mcp/internal/config"
	"github.com/neo4j/mcp/internal/database/mocks"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"go.uber.org/mock/gomock"
)

func TestExecuteGDSFunctionHandler(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDBService := mocks.NewMockDatabaseService(ctrl)
	cfg := &config.Config{
		Database: "neo4j",
	}

	// Create test records for GDS function results
	testRecords := []*neo4j.Record{
		{
			Keys:   []string{"nodeId", "score"},
			Values: []interface{}{1, 0.15},
		},
		{
			Keys:   []string{"nodeId", "score"},
			Values: []interface{}{2, 0.25},
		},
	}

	tests := []struct {
		name           string
		input          ExecuteGDSFunctionInput
		expectFunction bool
		expectCleanup  bool
		wantErr        bool
		errorMessage   string
	}{
		{
			name: "successful GDS function execution with projection name",
			input: ExecuteGDSFunctionInput{
				FunctionName: "gds.pageRank.stream",
				FunctionParams: map[string]interface{}{
					"dampingFactor": 0.85,
				},
				ProjectionName: stringPtr("testGraph"),
			},
			expectFunction: true,
			expectCleanup:  true,
			wantErr:        false,
		},
		{
			name: "successful GDS function execution with complex params",
			input: ExecuteGDSFunctionInput{
				FunctionName: "gds.louvain.stream",
				FunctionParams: map[string]interface{}{
					"maxIterations":                  10,
					"includeIntermediateCommunities": true,
					"concurrency":                    4,
				},
				ProjectionName: stringPtr("socialNetwork"),
			},
			expectFunction: true,
			expectCleanup:  true,
			wantErr:        false,
		},
		{
			name: "error when no projection name provided",
			input: ExecuteGDSFunctionInput{
				FunctionName: "gds.pageRank.stream",
				FunctionParams: map[string]interface{}{
					"dampingFactor": 0.85,
				},
				ProjectionName: stringPtr(""),
			},
			expectFunction: false,
			expectCleanup:  false,
			wantErr:        true,
			errorMessage:   "A projection name is required",
		},
		{
			name: "error when projection name is nil",
			input: ExecuteGDSFunctionInput{
				FunctionName: "gds.pageRank.stream",
				FunctionParams: map[string]interface{}{
					"dampingFactor": 0.85,
				},
				ProjectionName: nil,
			},
			expectFunction: false,
			expectCleanup:  false,
			wantErr:        true,
			errorMessage:   "A projection name is required",
		},
		{
			name: "error when GDS function execution fails",
			input: ExecuteGDSFunctionInput{
				FunctionName: "gds.invalidFunction.stream",
				FunctionParams: map[string]interface{}{
					"dampingFactor": 0.85,
				},
				ProjectionName: stringPtr("testGraph"),
			},
			expectFunction: true,
			expectCleanup:  true,
			wantErr:        true,
			errorMessage:   "Failed to execute GDS function",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.expectFunction {
				if tt.wantErr && tt.errorMessage == "Failed to execute GDS function" {
					// Mock function execution failure
					mockDBService.EXPECT().
						ExecuteWriteQuery(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
						Return(nil, errors.New("GDS function execution failed")).
						Times(1)
				} else {
					// Mock successful function execution
					mockDBService.EXPECT().
						ExecuteWriteQuery(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
						Return(testRecords, nil).
						Times(1)
				}
			}

			if tt.expectCleanup && !tt.wantErr {
				// Mock successful cleanup (only if function execution succeeded)
				mockDBService.EXPECT().
					ExecuteWriteQuery(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Return([]*neo4j.Record{}, nil).
					Times(1)
			} else if tt.expectCleanup && tt.wantErr {
				// Mock cleanup even after function execution failure
				mockDBService.EXPECT().
					ExecuteWriteQuery(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Return([]*neo4j.Record{}, nil).
					Times(1)
			}

			handler := ExecuteGDSFunctionHandler(&ToolDependencies{
				Config:    cfg,
				DBService: mockDBService,
			})

			request := mcp.CallToolRequest{
				Params: mcp.CallToolParams{
					Arguments: map[string]interface{}{
						"function_name":   tt.input.FunctionName,
						"function_params": tt.input.FunctionParams,
					},
				},
			}

			// Add projection name if provided
			if tt.input.ProjectionName != nil {
				args := request.Params.Arguments.(map[string]interface{})
				args["projection_name"] = *tt.input.ProjectionName
			}

			result, err := handler(context.Background(), request)

			if tt.wantErr {
				if err != nil {
					t.Errorf("Expected error but got none")
				}
				if result != nil && !result.IsError {
					t.Errorf("Expected error result but got success")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if result == nil {
				t.Errorf("Expected result but got nil")
				return
			}

			if result.IsError {
				t.Errorf("Expected success result but got error")
				return
			}

			if len(result.Content) == 0 {
				t.Errorf("Expected non-empty content result")
				return
			}
		})
	}
}

func TestExecuteGDSFunctionHandler_EdgeCases(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDBService := mocks.NewMockDatabaseService(ctrl)
	cfg := &config.Config{
		Database: "neo4j",
	}

	t.Run("nil database service", func(t *testing.T) {
		handler := ExecuteGDSFunctionHandler(&ToolDependencies{
			Config:    cfg,
			DBService: nil,
		})

		request := mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Arguments: map[string]interface{}{
					"function_name":   "gds.pageRank.stream",
					"function_params": map[string]interface{}{"dampingFactor": 0.85},
					"projection_name": "testGraph",
				},
			},
		}

		result, err := handler(context.Background(), request)

		if err != nil {
			t.Errorf("Expected error but got none")
		}
		if result != nil && !result.IsError {
			t.Errorf("Expected error result but got success")
		}
	})

	t.Run("invalid request binding", func(t *testing.T) {
		handler := ExecuteGDSFunctionHandler(&ToolDependencies{
			Config:    cfg,
			DBService: mockDBService,
		})

		// Create invalid request with missing required fields
		request := mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Arguments: map[string]interface{}{
					"invalid_field": "invalid_value",
				},
			},
		}

		result, err := handler(context.Background(), request)

		if err != nil {
			t.Errorf("Expected error but got none")
		}
		if result != nil && !result.IsError {
			t.Errorf("Expected error result but got success")
		}
	})
}

func TestGDSExecutionResult_Structure(t *testing.T) {
	result := GDSExecutionResult{
		ProjectionName: "testGraph",
		FunctionName:   "gds.pageRank.stream",
		FunctionParams: map[string]interface{}{
			"dampingFactor": 0.85,
		},
		Results: []map[string]interface{}{
			{"nodeId": 1, "score": 0.15},
			{"nodeId": 2, "score": 0.25},
		},
		ExecutionTime: "150ms",
	}

	if result.ProjectionName != "testGraph" {
		t.Error("ProjectionName should be preserved")
	}

	if result.FunctionName != "gds.pageRank.stream" {
		t.Error("FunctionName should be preserved")
	}

	if result.FunctionParams["dampingFactor"] != 0.85 {
		t.Error("FunctionParams should be preserved")
	}

	if len(result.Results) != 2 {
		t.Error("Results should contain 2 items")
	}

	if result.ExecutionTime != "150ms" {
		t.Error("ExecutionTime should be preserved")
	}
}

func TestGenerateDefaultProjection(t *testing.T) {
	projectionName := "testGraph"
	projection := generateDefaultProjection(projectionName)

	if !contains(projection, projectionName) {
		t.Errorf("Generated projection should contain projection name '%s'", projectionName)
	}

	if !contains(projection, "gds.graph.project") {
		t.Error("Generated projection should contain gds.graph.project call")
	}

	if !contains(projection, "*") {
		t.Error("Default projection should include all nodes and relationships (*)")
	}
}

func TestGenerateSpecificProjection(t *testing.T) {
	projectionName := "testGraph"
	nodeLabels := []string{"Person", "Company"}
	relationshipTypes := []string{"KNOWS", "WORKS_FOR"}

	projection := generateSpecificProjection(projectionName, nodeLabels, relationshipTypes)

	if !contains(projection, projectionName) {
		t.Errorf("Generated projection should contain projection name '%s'", projectionName)
	}

	if !contains(projection, "gds.graph.project") {
		t.Error("Generated projection should contain gds.graph.project call")
	}

	for _, label := range nodeLabels {
		if !contains(projection, label) {
			t.Errorf("Generated projection should contain node label '%s'", label)
		}
	}

	for _, relType := range relationshipTypes {
		if !contains(projection, relType) {
			t.Errorf("Generated projection should contain relationship type '%s'", relType)
		}
	}
}

// Helper functions

func contains(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr ||
			len(s) > len(substr) &&
				(s[:len(substr)] == substr ||
					s[len(s)-len(substr):] == substr ||
					findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
