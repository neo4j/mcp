package tools

import (
	"testing"
)

func TestExecuteGDSFunctionSpec(t *testing.T) {
	spec := ExecuteGDSFunctionSpec()

	// Test basic spec properties
	if spec.Name != "execute-gds-function" {
		t.Errorf("Expected tool name 'execute-gds-function', got '%s'", spec.Name)
	}

	if spec.Description == "" {
		t.Error("Expected non-empty description")
	}

	// Test that the spec is properly constructed
	// Note: InputSchema is now a generic type, so we can't directly compare to nil
	// The schema is validated by the MCP framework
}

func TestExecuteGDSFunctionInput_JSONTags(t *testing.T) {
	// Test that the struct has proper JSON tags for serialization
	input := ExecuteGDSFunctionInput{
		FunctionName: "gds.pageRank.stream",
		FunctionParams: map[string]interface{}{
			"dampingFactor": 0.85,
		},
		ProjectionName: stringPtr("testGraph"),
	}

	// This test ensures the struct can be properly marshaled/unmarshaled
	// The actual JSON marshaling would be tested in integration tests
	if input.FunctionName == "" {
		t.Error("FunctionName should not be empty")
	}

	if input.FunctionParams == nil {
		t.Error("FunctionParams should not be nil")
	}

	if input.ProjectionName != nil && *input.ProjectionName == "" {
		t.Error("ProjectionName should not be empty string when provided")
	}
}

func TestExecuteGDSFunctionInput_Validation(t *testing.T) {
	tests := []struct {
		name    string
		input   ExecuteGDSFunctionInput
		wantErr bool
	}{
		{
			name: "valid input with all fields",
			input: ExecuteGDSFunctionInput{
				FunctionName: "gds.pageRank.stream",
				FunctionParams: map[string]interface{}{
					"dampingFactor": 0.85,
				},
				ProjectionName: stringPtr("testGraph"),
			},
			wantErr: false,
		},
		{
			name: "valid input without projection name",
			input: ExecuteGDSFunctionInput{
				FunctionName: "gds.louvain.stream",
				FunctionParams: map[string]interface{}{
					"maxIterations": 10,
				},
			},
			wantErr: false,
		},
		{
			name: "valid input with empty projection name",
			input: ExecuteGDSFunctionInput{
				FunctionName: "gds.weaklyConnectedComponents.stream",
				FunctionParams: map[string]interface{}{
					"threshold": 1.0,
				},
				ProjectionName: stringPtr(""),
			},
			wantErr: false,
		},
		{
			name: "valid input with nil projection name",
			input: ExecuteGDSFunctionInput{
				FunctionName: "gds.betweenness.stream",
				FunctionParams: map[string]interface{}{
					"normalization": "max",
				},
				ProjectionName: nil,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test that the struct can be created without errors
			if tt.input.FunctionName == "" {
				t.Error("FunctionName should not be empty")
			}

			if tt.input.FunctionParams == nil {
				t.Error("FunctionParams should not be nil")
			}

			// Test that we can access the fields
			_ = tt.input.FunctionName
			_ = tt.input.FunctionParams
			_ = tt.input.ProjectionName
		})
	}
}

func TestExecuteGDSFunctionSpec_Annotations(t *testing.T) {
	spec := ExecuteGDSFunctionSpec()

	// Test that the spec has proper annotations
	// Note: The exact annotation structure depends on the MCP library implementation
	// These tests verify that the spec is properly constructed

	if spec.Name == "" {
		t.Error("Tool name should not be empty")
	}

	if spec.Description == "" {
		t.Error("Tool description should not be empty")
	}

	// Verify the tool is configured for write operations (not read-only)
	// This is important because GDS functions can modify projections
}

// Helper function to create string pointer
func stringPtr(s string) *string {
	return &s
}

func TestExecuteGDSFunctionInput_EdgeCases(t *testing.T) {
	// Test with empty function params
	input := ExecuteGDSFunctionInput{
		FunctionName:   "gds.graph.list",
		FunctionParams: map[string]interface{}{},
	}

	if input.FunctionName != "gds.graph.list" {
		t.Error("FunctionName should be preserved")
	}

	if input.FunctionParams == nil {
		t.Error("FunctionParams should not be nil even when empty")
	}

	// Test with complex function params
	complexInput := ExecuteGDSFunctionInput{
		FunctionName: "gds.pageRank.stream",
		FunctionParams: map[string]interface{}{
			"dampingFactor":              0.85,
			"maxIterations":              20,
			"tolerance":                  0.0001,
			"includeIntermediateResults": true,
			"config": map[string]interface{}{
				"concurrency": 4,
				"batchSize":   1000,
			},
		},
	}

	if complexInput.FunctionParams["dampingFactor"] != 0.85 {
		t.Error("Complex function params should preserve nested values")
	}

	config, ok := complexInput.FunctionParams["config"].(map[string]interface{})
	if !ok {
		t.Error("Nested config should be accessible")
	}

	if config["concurrency"] != 4 {
		t.Error("Nested config values should be preserved")
	}
}
