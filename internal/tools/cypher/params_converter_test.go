package cypher_test

import (
	"encoding/json"
	"testing"

	"github.com/neo4j/mcp/internal/tools/cypher"
	"github.com/stretchr/testify/assert"
)

func TestBindArgs(t *testing.T) {
	// Define a target struct for unmarshaling
	type TargetStruct struct {
		Query  string         `json:"query"`
		Params map[string]any `json:"params"`
	}

	tests := []struct {
		name    string
		request any
		expected TargetStruct
		expectErr bool
	}{
		{
			name: "basic integer parameter",
			request: map[string]any{
				"query":  "MATCH (n) WHERE n.id = $id RETURN n",
				"params": map[string]any{"id": 1},
			},
			expected: TargetStruct{
				Query:  "MATCH (n) WHERE n.id = $id RETURN n",
				Params: map[string]any{"id": int64(1)},
			},
		},
		{
			name: "basic float parameter",
			request: map[string]any{
				"query":  "MATCH (n) WHERE n.value = $value RETURN n",
				"params": map[string]any{"value": 1.5},
			},
			expected: TargetStruct{
				Query:  "MATCH (n) WHERE n.value = $value RETURN n",
				Params: map[string]any{"value": float64(1.5)},
			},
		},
		{
			name: "float as whole number should become int",
			request: map[string]any{
				"query":  "MATCH (n) WHERE n.limit = $limit RETURN n",
				"params": map[string]any{"limit": 1.0},
			},
			expected: TargetStruct{
				Query:  "MATCH (n) WHERE n.limit = $limit RETURN n",
				Params: map[string]any{"limit": int64(1)},
			},
		},
		{
			name: "mixed parameters",
			request: map[string]any{
				"query":  "MATCH (n) WHERE n.id = $id AND n.value = $value RETURN n",
				"params": map[string]any{"id": 1, "value": 2.5},
			},
			expected: TargetStruct{
				Query:  "MATCH (n) WHERE n.id = $id AND n.value = $value RETURN n",
				Params: map[string]any{"id": int64(1), "value": float64(2.5)},
			},
		},
		{
			name: "nested map with numbers",
			request: map[string]any{
				"query": "MATCH (n) WHERE n.data = $data RETURN n",
				"params": map[string]any{
					"data": map[string]any{
						"count": 10,
						"ratio": 0.5,
						"threshold": 5.0,
					},
				},
			},
			expected: TargetStruct{
				Query: "MATCH (n) WHERE n.data = $data RETURN n",
				Params: map[string]any{
					"data": map[string]any{
						"count": int64(10),
						"ratio": float64(0.5),
						"threshold": int64(5),
					},
				},
			},
		},
		{
			name: "slice with numbers",
			request: map[string]any{
				"query":  "MATCH (n) WHERE n.list IN $list RETURN n",
				"params": map[string]any{"list": []any{1, 2.0, 3.5}},
			},
			expected: TargetStruct{
				Query:  "MATCH (n) WHERE n.list IN $list RETURN n",
				Params: map[string]any{"list": []any{int64(1), int64(2), float64(3.5)}},
			},
		},
		{
			name: "empty params",
			request: map[string]any{
				"query":  "MATCH (n) RETURN n",
				"params": map[string]any{},
			},
			expected: TargetStruct{
				Query:  "MATCH (n) RETURN n",
				Params: map[string]any{},
			},
		},
		{
			name: "no params field",
			request: map[string]any{
				"query": "MATCH (n) RETURN n",
			},
			expected: TargetStruct{
				Query:  "MATCH (n) RETURN n",
				Params: nil,
			},
		},
		{
			name: "non-map request should error",
			request:   "invalid request string",
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var target TargetStruct
			err := cypher.BindArgs(tt.request, &target)

			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected.Query, target.Query)
				assert.Equal(t, tt.expected.Params, target.Params)
			}
		})
	}
}

func TestConvertNumbers(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		expected any
	}{
		{
			name:     "float64 to int64",
			input:    json.Number("123"),
			expected: int64(123),
		},
		{
			name:     "float64 to float64",
			input:    json.Number("123.45"),
			expected: float64(123.45),
		},
		{
			name:     "string",
			input:    "hello",
			expected: "hello",
		},
		{
			name: "map with mixed numbers",
			input: map[string]any{
				"intVal":   json.Number("10"),
				"floatVal": json.Number("20.5"),
				"strVal":   "test",
				"wholeFloat": json.Number("30.0"),
			},
			expected: map[string]any{
				"intVal":   int64(10),
				"floatVal": float64(20.5),
				"strVal":   "test",
				"wholeFloat": int64(30),
			},
		},
		{
			name: "slice with mixed numbers",
			input: []any{
				json.Number("1"),
				json.Number("2.0"),
				json.Number("3.5"),
				"text",
			},
			expected: []any{
				int64(1),
				int64(2),
				float64(3.5),
				"text",
			},
		},
		{
			name:     "nil input",
			input:    nil,
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cypher.ConvertNumbers(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}


