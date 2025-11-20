package cypher_test

import (
	"encoding/json"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/neo4j/mcp/internal/tools/cypher"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBindArgumentsWithReadCypherInput(t *testing.T) {
	tests := []struct {
		name       string
		arguments  map[string]any
		wantQuery  string
		wantParams cypher.CypherParams
		wantErr    bool
	}{
		{
			name: "basic integer parameter",
			arguments: map[string]any{
				"query":  "MATCH (n) WHERE n.id = $id RETURN n",
				"params": cypher.CypherParams{"id": 1},
			},
			wantQuery:  "MATCH (n) WHERE n.id = $id RETURN n",
			wantParams: cypher.CypherParams{"id": int64(1)},
		},
		{
			name: "basic float parameter",
			arguments: map[string]any{
				"query":  "MATCH (n) WHERE n.value = $value RETURN n",
				"params": cypher.CypherParams{"value": 1.5},
			},
			wantQuery:  "MATCH (n) WHERE n.value = $value RETURN n",
			wantParams: cypher.CypherParams{"value": float64(1.5)},
		},
		{
			name: "float as whole number should become int",
			arguments: map[string]any{
				"query":  "MATCH (n) WHERE n.limit = $limit RETURN n",
				"params": cypher.CypherParams{"limit": 1.0},
			},
			wantQuery:  "MATCH (n) WHERE n.limit = $limit RETURN n",
			wantParams: cypher.CypherParams{"limit": int64(1)},
		},
		{
			name: "mixed parameters",
			arguments: map[string]any{
				"query":  "MATCH (n) WHERE n.id = $id AND n.value = $value RETURN n",
				"params": cypher.CypherParams{"id": 1, "value": 2.5},
			},
			wantQuery:  "MATCH (n) WHERE n.id = $id AND n.value = $value RETURN n",
			wantParams: cypher.CypherParams{"id": int64(1), "value": float64(2.5)},
		},
		{
			name: "nested map with numbers",
			arguments: map[string]any{
				"query": "MATCH (n) WHERE n.data = $data RETURN n",
				"params": cypher.CypherParams{
					"data": map[string]any{
						"count":     10,
						"ratio":     0.5,
						"threshold": 5.0,
					},
				},
			},
			wantQuery: "MATCH (n) WHERE n.data = $data RETURN n",
			wantParams: cypher.CypherParams{
				"data": map[string]any{
					"count":     int64(10),
					"ratio":     float64(0.5),
					"threshold": int64(5),
				},
			},
		},
		{
			name: "slice with mixed numbers",
			arguments: map[string]any{
				"query":  "MATCH (n) WHERE n.list IN $list RETURN n",
				"params": cypher.CypherParams{"list": []any{1, 2.0, 3.5}},
			},
			wantQuery:  "MATCH (n) WHERE n.list IN $list RETURN n",
			wantParams: cypher.CypherParams{"list": []any{int64(1), int64(2), float64(3.5)}},
		},
		{
			name: "deeply nested structure",
			arguments: map[string]any{
				"query": "MATCH (n) WHERE n.complex = $complex RETURN n",
				"params": cypher.CypherParams{
					"complex": map[string]any{
						"level1": map[string]any{
							"level2": []any{
								map[string]any{
									"value": 42,
									"ratio": 0.75,
								},
								1.0,
								2.5,
							},
						},
					},
				},
			},
			wantQuery: "MATCH (n) WHERE n.complex = $complex RETURN n",
			wantParams: cypher.CypherParams{
				"complex": map[string]any{
					"level1": map[string]any{
						"level2": []any{
							map[string]any{
								"value": int64(42),
								"ratio": float64(0.75),
							},
							int64(1),
							float64(2.5),
						},
					},
				},
			},
		},
		{
			name: "empty params map",
			arguments: map[string]any{
				"query":  "MATCH (n) RETURN n",
				"params": cypher.CypherParams{},
			},
			wantQuery:  "MATCH (n) RETURN n",
			wantParams: cypher.CypherParams{},
		},
		{
			name: "no params field (nil params)",
			arguments: map[string]any{
				"query": "MATCH (n) RETURN n",
			},
			wantQuery:  "MATCH (n) RETURN n",
			wantParams: nil,
		},
		{
			name: "string parameters should remain strings",
			arguments: map[string]any{
				"query":  "MATCH (n) WHERE n.name = $name RETURN n",
				"params": cypher.CypherParams{"name": "Alice"},
			},
			wantQuery:  "MATCH (n) WHERE n.name = $name RETURN n",
			wantParams: cypher.CypherParams{"name": "Alice"},
		},
		{
			name: "boolean parameters should remain booleans",
			arguments: map[string]any{
				"query":  "MATCH (n) WHERE n.active = $active RETURN n",
				"params": cypher.CypherParams{"active": true},
			},
			wantQuery:  "MATCH (n) WHERE n.active = $active RETURN n",
			wantParams: cypher.CypherParams{"active": true},
		},
		{
			name: "null parameters should remain nil",
			arguments: map[string]any{
				"query":  "MATCH (n) WHERE n.optional = $optional RETURN n",
				"params": cypher.CypherParams{"optional": nil},
			},
			wantQuery:  "MATCH (n) WHERE n.optional = $optional RETURN n",
			wantParams: cypher.CypherParams{"optional": nil},
		},
		{
			name: "large integer should become int64",
			arguments: map[string]any{
				"query":  "MATCH (n) WHERE n.bignum = $bignum RETURN n",
				"params": cypher.CypherParams{"bignum": 1000000000000000000}, // 10^18
			},
			wantQuery:  "MATCH (n) WHERE n.bignum = $bignum RETURN n",
			wantParams: cypher.CypherParams{"bignum": int64(1000000000000000000)},
		},
		{
			name: "zero should become int64",
			arguments: map[string]any{
				"query":  "MATCH (n) LIMIT $limit",
				"params": cypher.CypherParams{"limit": 0},
			},
			wantQuery:  "MATCH (n) LIMIT $limit",
			wantParams: cypher.CypherParams{"limit": int64(0)},
		},
		{
			name: "zero point zero should become int64",
			arguments: map[string]any{
				"query":  "MATCH (n) LIMIT $limit",
				"params": cypher.CypherParams{"limit": 0.0},
			},
			wantQuery:  "MATCH (n) LIMIT $limit",
			wantParams: cypher.CypherParams{"limit": int64(0)},
		},
		{
			name: "negative numbers",
			arguments: map[string]any{
				"query":  "MATCH (n) WHERE n.balance = $balance AND n.adjustment = $adjustment RETURN n",
				"params": cypher.CypherParams{"balance": -100, "adjustment": -1.5},
			},
			wantQuery:  "MATCH (n) WHERE n.balance = $balance AND n.adjustment = $adjustment RETURN n",
			wantParams: cypher.CypherParams{"balance": int64(-100), "adjustment": float64(-1.5)},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := mcp.CallToolRequest{
				Params: mcp.CallToolParams{
					Arguments: tt.arguments,
				},
			}

			var args cypher.ReadCypherInput
			err := request.BindArguments(&args)

			if tt.wantErr {
				assert.Error(t, err, "expected error but got none")
				return
			}

			require.NoError(t, err, "bindArguments should not error")
			assert.Equal(t, tt.wantQuery, args.Query, "query mismatch")
			assert.Equal(t, tt.wantParams, args.Params, "params mismatch")
		})
	}
}

func TestBindArgumentsWithWriteCypherInput(t *testing.T) {
	tests := []struct {
		name       string
		arguments  map[string]any
		wantQuery  string
		wantParams cypher.CypherParams
	}{
		{
			name: "write query with integer parameter",
			arguments: map[string]any{
				"query":  "CREATE (n:Node) SET n.count = $count RETURN n",
				"params": cypher.CypherParams{"count": 5},
			},
			wantQuery:  "CREATE (n:Node) SET n.count = $count RETURN n",
			wantParams: cypher.CypherParams{"count": int64(5)},
		},
		{
			name: "write query with mixed parameters",
			arguments: map[string]any{
				"query":  "CREATE (n:Node {name: $name, score: $score}) RETURN n",
				"params": cypher.CypherParams{"name": "test", "score": 42.0},
			},
			wantQuery:  "CREATE (n:Node {name: $name, score: $score}) RETURN n",
			wantParams: cypher.CypherParams{"name": "test", "score": int64(42)},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := mcp.CallToolRequest{
				Params: mcp.CallToolParams{
					Arguments: tt.arguments,
				},
			}

			var args cypher.WriteCypherInput
			err := request.BindArguments(&args)

			require.NoError(t, err, "bindArguments should not error")
			assert.Equal(t, tt.wantQuery, args.Query, "query mismatch")
			assert.Equal(t, tt.wantParams, args.Params, "params mismatch")
		})
	}
}

func TestBindArgumentsErrorHandling(t *testing.T) {
	t.Run("invalid arguments type - string instead of map", func(t *testing.T) {
		request := mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Arguments: "invalid string instead of map",
			},
		}

		var args cypher.ReadCypherInput
		err := request.BindArguments(&args)

		assert.Error(t, err, "should error on invalid argument type")
	})

	t.Run("invalid arguments type - array instead of map", func(t *testing.T) {
		request := mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Arguments: []any{"invalid", "array"},
			},
		}

		var args cypher.ReadCypherInput
		err := request.BindArguments(&args)

		assert.Error(t, err, "should error on invalid argument type")
	})

	t.Run("missing query field should have empty query", func(t *testing.T) {
		request := mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Arguments: map[string]any{
					"params": map[string]any{"id": 1},
				},
			},
		}

		var args cypher.ReadCypherInput
		err := request.BindArguments(&args)

		require.NoError(t, err)
		assert.Equal(t, "", args.Query, "query should be empty when not provided")
	})
}

func TestConvertNumbers(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		expected any
	}{
		{
			name:     "integer from json.Number",
			input:    json.Number("42"),
			expected: int64(42),
		},
		{
			name:     "float from json.Number",
			input:    json.Number("3.14"),
			expected: float64(3.14),
		},
		{
			name:     "whole number float from json.Number becomes int",
			input:    json.Number("10.0"),
			expected: float64(10),
		},
		{
			name:     "zero from json.Number",
			input:    json.Number("0"),
			expected: int64(0),
		},
		{
			name:     "zero point zero from json.Number becomes int",
			input:    json.Number("0.0"),
			expected: float64(0),
		},
		{
			name:     "negative integer from json.Number",
			input:    json.Number("-42"),
			expected: int64(-42),
		},
		{
			name:     "negative float from json.Number",
			input:    json.Number("-3.14"),
			expected: float64(-3.14),
		},
		{
			name:     "string is preserved",
			input:    "hello",
			expected: "hello",
		},
		{
			name:     "boolean is preserved",
			input:    true,
			expected: true,
		},
		{
			name:     "nil is preserved",
			input:    nil,
			expected: nil,
		},
		{
			name: "map with numbers",
			input: map[string]any{
				"count": json.Number("10"),
				"ratio": json.Number("0.5"),
				"name":  "test",
			},
			expected: map[string]any{
				"count": int64(10),
				"ratio": float64(0.5),
				"name":  "test",
			},
		},
		{
			name: "nested map with numbers",
			input: map[string]any{
				"outer": map[string]any{
					"inner": json.Number("42"),
				},
			},
			expected: map[string]any{
				"outer": map[string]any{
					"inner": int64(42),
				},
			},
		},
		{
			name:     "slice with numbers",
			input:    []any{json.Number("1"), json.Number("2.0"), json.Number("3.5")},
			expected: []any{int64(1), float64(2), float64(3.5)},
		},
		{
			name: "map with slice containing maps",
			input: map[string]any{
				"items": []any{
					map[string]any{
						"id":    json.Number("1"),
						"score": json.Number("9.5"),
					},
					map[string]any{
						"id":    json.Number("2"),
						"score": json.Number("10.0"),
					},
				},
			},
			expected: map[string]any{
				"items": []any{
					map[string]any{
						"id":    int64(1),
						"score": float64(9.5),
					},
					map[string]any{
						"id":    int64(2),
						"score": float64(10),
					},
				},
			},
		},
		{
			name:     "empty map",
			input:    map[string]any{},
			expected: map[string]any{},
		},
		{
			name:     "empty slice",
			input:    []any{},
			expected: []any{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cypher.ConvertNumbers(tt.input)
			assert.Equal(t, tt.expected, result, "ConvertNumbers should produce expected output")
		})
	}
}

// TestLimitScenario tests the specific scenario from issue #70
func TestLimitScenario(t *testing.T) {
	// This is the exact scenario that was failing before the fix
	request := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: map[string]any{
				"query":  "MATCH(n) RETURN n LIMIT $limit",
				"params": map[string]any{"limit": 1},
			},
		},
	}

	var args cypher.ReadCypherInput
	err := request.BindArguments(&args)

	require.NoError(t, err)
	assert.Equal(t, "MATCH(n) RETURN n LIMIT $limit", args.Query)

	// The critical assertion: limit must be int64, not float64
	limitValue, ok := args.Params["limit"]
	require.True(t, ok, "limit should be in params")

	// Verify it's int64, not float64
	intVal, ok := limitValue.(int64)
	assert.True(t, ok, "limit should be int64, not %T", limitValue)
	assert.Equal(t, int64(1), intVal)
}
