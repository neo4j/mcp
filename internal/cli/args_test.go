package cli

import (
	"os"
	"testing"
)

func TestHandleArgs(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		version  string
		expected bool
	}{
		{
			name:     "no flags",
			args:     []string{"neo4j-mcp"},
			version:  "1.0.0",
			expected: false,
		},
		{
			name:     "version flag short form",
			args:     []string{"neo4j-mcp", "-v"},
			version:  "1.0.0",
			expected: true,
		},
		{
			name:     "version flag long form",
			args:     []string{"neo4j-mcp", "--version"},
			version:  "1.0.0",
			expected: true,
		},
		{
			name:     "help flag short form",
			args:     []string{"neo4j-mcp", "-h"},
			version:  "1.0.0",
			expected: true,
		},
		{
			name:     "help flag long form",
			args:     []string{"neo4j-mcp", "--help"},
			version:  "1.0.0",
			expected: true,
		},
		{
			name:     "unknown flag",
			args:     []string{"neo4j-mcp", "-x"},
			version:  "1.0.0",
			expected: false,
		},
		{
			name:     "multiple arguments with flag",
			args:     []string{"neo4j-mcp", "-v", "extra"},
			version:  "1.0.0",
			expected: true,
		},
		{
			name:     "version and help flags together",
			args:     []string{"neo4j-mcp", "-v", "-h"},
			version:  "1.0.0",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			originalArgs := os.Args
			defer func() { os.Args = originalArgs }()

			os.Args = tt.args
			result := HandleArgs(tt.version)

			if result != tt.expected {
				t.Errorf("HandleArgs() = %v, want %v", result, tt.expected)
			}
		})
	}
}
