//
// Copyright (c) "Neo4j"
// Neo4j Sweden AB [http://neo4j.com]

package cypher_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	analytics "github.com/neo4j/mcp/internal/analytics/mocks"
	db "github.com/neo4j/mcp/internal/database/mocks"
	"github.com/neo4j/mcp/internal/tools"
	"github.com/neo4j/mcp/internal/tools/cypher"
	"github.com/neo4j/neo4j-go-driver/v6/neo4j"
	"go.uber.org/mock/gomock"
)

func TestGetSchemaHandler(t *testing.T) {
	ctrl := gomock.NewController(t)
	analyticsService := analytics.NewMockService(ctrl)
	defer ctrl.Finish()

	t.Run("successful schema retrieval", func(t *testing.T) {
		mockDB := db.NewMockService(ctrl)
		mockDB.EXPECT().
			ExecuteReadQuery(gomock.Any(), gomock.Any(), gomock.Eq(map[string]any{"sampleSize": int32(100)})).
			Return([]*neo4j.Record{
				{
					Keys: []string{"key", "value"},
					Values: []any{
						"Movie",
						map[string]any{
							"type": "node",
							"properties": map[string]any{
								"title": map[string]any{"type": "STRING", "indexed": false},
							},
							"relationships": map[string]any{
								"ACTED_IN": map[string]any{
									"count":      1,
									"direction":  "in",
									"labels":     []any{"Person"},
									"properties": map[string]any{},
								},
							},
						},
					},
				},
				{
					Keys: []string{"key", "value"},
					Values: []any{
						"ACTED_IN",
						map[string]any{
							"type":       "relationship",
							"properties": map[string]any{},
						},
					},
				},
			}, nil)

		deps := &tools.ToolDependencies{
			DBService:        mockDB,
			AnalyticsService: analyticsService,
		}

		handler := cypher.GetSchemaHandler(deps, 100)
		result, err := handler(context.Background(), mcp.CallToolRequest{})

		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
		if result == nil || result.IsError {
			t.Error("Expected success result")
		}
	})

	t.Run("database query failure", func(t *testing.T) {
		mockDB := db.NewMockService(ctrl)
		mockDB.EXPECT().
			ExecuteReadQuery(gomock.Any(), gomock.Any(), gomock.Any()).
			Return(nil, errors.New("connection failed"))

		deps := &tools.ToolDependencies{
			DBService:        mockDB,
			AnalyticsService: analyticsService,
		}

		handler := cypher.GetSchemaHandler(deps, 100)
		result, err := handler(context.Background(), mcp.CallToolRequest{})

		if err != nil {
			t.Errorf("Expected no error from handler, got: %v", err)
		}
		if result == nil || !result.IsError {
			t.Error("Expected error result")
		}
	})

	t.Run("nil database service", func(t *testing.T) {
		deps := &tools.ToolDependencies{
			DBService:        nil,
			AnalyticsService: analyticsService,
		}

		handler := cypher.GetSchemaHandler(deps, 100)
		result, err := handler(context.Background(), mcp.CallToolRequest{})

		if err != nil {
			t.Errorf("Expected no error from handler, got: %v", err)
		}
		if result == nil || !result.IsError {
			t.Error("Expected error result for nil database service")
		}
	})

	t.Run("No records returned from apoc query (empty database)", func(t *testing.T) {
		mockDB := db.NewMockService(ctrl)
		mockDB.EXPECT().
			ExecuteReadQuery(gomock.Any(), gomock.Any(), gomock.Any()).
			Return([]*neo4j.Record{}, nil)

		deps := &tools.ToolDependencies{
			DBService:        mockDB,
			AnalyticsService: analyticsService,
		}

		handler := cypher.GetSchemaHandler(deps, 100)
		result, err := handler(context.Background(), mcp.CallToolRequest{})

		if err != nil {
			t.Errorf("Expected no error from handler, got: %v", err)
		}

		if result == nil {
			t.Error("Expected non-nil result")
			return
		}

		if result.IsError {
			t.Error("Expected success result, not error")
			return
		}

		textContent := result.Content[0].(mcp.TextContent)
		if textContent.Text != "The get-schema tool executed successfully; however, since the Neo4j instance contains no data, no schema information was returned." {
			t.Error("Expected result content to be present for empty database case")
		}
	})

}

func TestGetSchemaProcessing(t *testing.T) {
	ctrl := gomock.NewController(t)
	analyticsService := analytics.NewMockService(ctrl)
	// Note: Handlers no longer emit events directly - events are emitted via hooks in server.go
	defer ctrl.Finish()

	testCases := []struct {
		name         string
		expectedErr  bool
		mockRecords  []*neo4j.Record
		expectedJSON string
	}{
		{
			name:        "successful schema processing",
			expectedErr: false,
			mockRecords: []*neo4j.Record{
				{
					Keys: []string{"key", "value"},
					Values: []any{
						"Movie",
						map[string]any{
							"type": "node",
							"properties": map[string]any{
								"title":    map[string]any{"type": "STRING", "indexed": false},
								"released": map[string]any{"type": "INTEGER", "indexed": true},
							},
							"relationships": map[string]any{
								"ACTED_IN": map[string]any{
									"count":     16,
									"direction": "in",
									"labels":    []any{"Person"},
									"properties": map[string]any{
										"year": map[string]any{"type": "DATE", "indexed": false},
									},
								},
							},
						},
					},
				},
				{
					Keys: []string{"key", "value"},
					Values: []any{
						"ACTED_IN",
						map[string]any{
							"type": "relationship",
							"properties": map[string]any{
								"roles": map[string]any{"type": "LIST"},
							},
						},
					},
				},
			},
			expectedJSON: `[
				{
					"key": "Movie",
					"value": {
						"properties": {
							"released": "INTEGER",
							"title": "STRING"
						},
						"relationships": {
							"ACTED_IN": {
								"direction": "in",
								"labels": [
									"Person"
								],
								"properties": {
									"year": "DATE"
								}
							}
						},
						"type": "node"
					}
				},
				{
					"key": "ACTED_IN",
					"value": {
						"properties": {
							"roles": "LIST"
						},
						"type": "relationship"
					}
				}
			]`,
		},
		{
			name:        "schema with multiple nodes and varied relationships",
			expectedErr: false,
			mockRecords: []*neo4j.Record{
				{
					Keys: []string{"key", "value"},
					Values: []any{
						"Movie",
						map[string]any{
							"type": "node",
							"properties": map[string]any{
								"title":    map[string]any{"type": "STRING", "indexed": false},
								"released": map[string]any{"type": "INTEGER", "indexed": false},
							},
							"relationships": map[string]any{
								"ACTED_IN": map[string]any{
									"direction": "in", "labels": []any{"Person"}, "properties": map[string]any{"roles": map[string]any{"type": "LIST"}},
								},
								"DIRECTED": map[string]any{
									"direction": "in", "labels": []any{"Person"}, "properties": map[string]any{},
								},
							},
						},
					},
				},
				{
					Keys: []string{"key", "value"},
					Values: []any{
						"Person",
						map[string]any{
							"type":       "node",
							"properties": map[string]any{"name": map[string]any{"type": "STRING"}, "born": map[string]any{"type": "INTEGER"}},
							"relationships": map[string]any{
								"ACTED_IN": map[string]any{
									"direction": "out", "labels": []any{"Movie"}, "properties": map[string]any{"roles": map[string]any{"type": "LIST"}},
								},
								"DIRECTED": map[string]any{
									"direction": "out", "labels": []any{"Movie"}, "properties": map[string]any{},
								},
							},
						},
					},
				},
				{
					Keys:   []string{"key", "value"},
					Values: []any{"ACTED_IN", map[string]any{"type": "relationship", "properties": map[string]any{"roles": map[string]any{"type": "LIST"}}}},
				},
				{
					Keys:   []string{"key", "value"},
					Values: []any{"DIRECTED", map[string]any{"type": "relationship", "properties": map[string]any{}}},
				},
			},
			expectedJSON: `[
				{
					"key": "Movie",
					"value": {
						"properties": {"released": "INTEGER", "title": "STRING"},
						"relationships": {
							"ACTED_IN": {"direction": "in", "labels": ["Person"], "properties": {"roles": "LIST"}},
							"DIRECTED": {"direction": "in", "labels": ["Person"]}
						},
						"type": "node"
					}
				},
				{
					"key": "Person",
					"value": {
						"properties": {"born": "INTEGER", "name": "STRING"},
						"relationships": {
							"ACTED_IN": {"direction": "out", "labels": ["Movie"], "properties": {"roles": "LIST"}},
							"DIRECTED": {"direction": "out", "labels": ["Movie"]}
						},
						"type": "node"
					}
				},
				{
					"key": "ACTED_IN",
					"value": {"properties": {"roles": "LIST"}, "type": "relationship"}
				},
				{
					"key": "DIRECTED",
					"value": {"type": "relationship"}
				}
			]`,
		},
		{
			name:        "schema with a node with no relationships",
			expectedErr: false,
			mockRecords: []*neo4j.Record{
				{
					Keys: []string{"key", "value"},
					Values: []any{
						"Genre",
						map[string]any{
							"type":          "node",
							"properties":    map[string]any{"name": map[string]any{"type": "STRING"}},
							"relationships": map[string]any{},
						},
					},
				},
			},
			expectedJSON: `[
				{
					"key": "Genre",
					"value": {
						"properties": {"name": "STRING"},
						"type": "node"
					}
				}
			]`,
		},
		{
			name:        "schema with a node with no relationships (relationships nil)",
			expectedErr: false,
			mockRecords: []*neo4j.Record{
				{
					Keys: []string{"key", "value"},
					Values: []any{
						"Genre",
						map[string]any{
							"type":          "node",
							"properties":    map[string]any{"name": map[string]any{"type": "STRING"}},
							"relationships": nil,
						},
					},
				},
			},
			expectedJSON: `[
				{
					"key": "Genre",
					"value": {
						"properties": {"name": "STRING"},
						"type": "node"
					}
				}
			]`,
		},
		{
			name:        "should fail for invalid output returned by apoc.meta.schema (no key returned)",
			expectedErr: true,
			mockRecords: []*neo4j.Record{
				{
					Keys: []string{"value"},
					Values: []any{
						"Genre",
					},
				},
			},
			expectedJSON: `[]`,
		},
		{
			name:        "should fail for invalid output returned by apoc.meta.schema (invalid properties)",
			expectedErr: true,
			mockRecords: []*neo4j.Record{
				{
					Keys: []string{"key", "value"},
					Values: []any{
						"Genre",
						map[string]any{
							"type":          "node",
							"properties":    12,
							"relationships": map[string]any{},
						},
					},
				},
			},
			expectedJSON: `[]`,
		},
		{
			name:        "should fail for invalid output returned by apoc.meta.schema (invalid Node.Relationship.direction)",
			expectedErr: true,
			mockRecords: []*neo4j.Record{
				{
					Keys: []string{"key", "value"},
					Values: []any{
						"Movie",
						map[string]any{
							"type": "node",
							"properties": map[string]any{
								"title":    map[string]any{"type": "STRING", "indexed": false},
								"released": map[string]any{"type": "INTEGER", "indexed": false},
							},
							"relationships": map[string]any{
								"ACTED_IN": map[string]any{
									"direction": 12, "labels": []any{"Person"}, "properties": map[string]any{"roles": map[string]any{"type": "LIST"}},
								},
							},
						},
					},
				},
			},
			expectedJSON: `[]`,
		},
		{
			name:        "should fail for invalid output returned by apoc.meta.schema (invalid Node.relationship)",
			expectedErr: true,
			mockRecords: []*neo4j.Record{
				{
					Keys: []string{"key", "value"},
					Values: []any{
						"Movie",
						map[string]any{
							"type": "node",
							"properties": map[string]any{
								"title":    map[string]any{"type": "STRING", "indexed": false},
								"released": map[string]any{"type": "INTEGER", "indexed": false},
							},
							"relationships": map[string]any{
								"ACTED_IN": "something",
							},
						},
					},
				},
			},
			expectedJSON: `[]`,
		},
		{
			name:        "should fail for invalid output returned by apoc.meta.schema (invalid Node.relationship labels)",
			expectedErr: true,
			mockRecords: []*neo4j.Record{
				{
					Keys: []string{"key", "value"},
					Values: []any{
						"Movie",
						map[string]any{
							"type": "node",
							"properties": map[string]any{
								"title":    map[string]any{"type": "STRING", "indexed": false},
								"released": map[string]any{"type": "INTEGER", "indexed": false},
							},
							"relationships": map[string]any{
								"ACTED_IN": map[string]any{
									"direction": "in", "labels": "not-valid", "properties": map[string]any{
										"role": map[string]any{"type": "STRING", "indexed": false},
									},
								},
							},
						},
					},
				},
			},
			expectedJSON: `[]`,
		},
		{
			name:        "should fail for invalid output returned by apoc.meta.schema (invalid relationship properties)",
			expectedErr: true,
			mockRecords: []*neo4j.Record{
				{
					Keys: []string{"key", "value"},
					Values: []any{
						"ACTED_IN",
						map[string]any{
							"type":       "relationship",
							"labels":     []any{"Person"},
							"properties": "not-valid",
						},
					},
				},
			},
			expectedJSON: `[]`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockDB := db.NewMockService(ctrl)
			mockDB.EXPECT().
				ExecuteReadQuery(gomock.Any(), gomock.Any(), gomock.Any()).
				Return(tc.mockRecords, nil)

			deps := &tools.ToolDependencies{
				DBService:        mockDB,
				AnalyticsService: analyticsService,
			}

			handler := cypher.GetSchemaHandler(deps, 100)
			result, err := handler(context.Background(), mcp.CallToolRequest{})

			if err != nil {
				t.Fatalf("Expected no error, got: %v", err)
			}

			if result == nil || result.IsError {
				if tc.expectedErr {
					return
				}
				t.Fatal("Expected success result")
			}

			textContent, ok := result.Content[0].(mcp.TextContent)
			if !ok {
				t.Fatal("Expected result content to be TextContent")
			}

			var expectedData, actualData any
			if err := json.Unmarshal([]byte(tc.expectedJSON), &expectedData); err != nil {
				t.Fatalf("failed to unmarshal expected JSON: %v", err)
			}
			if err := json.Unmarshal([]byte(textContent.Text), &actualData); err != nil {
				t.Fatalf("failed to unmarshal actual JSON: %v", err)
			}

			expectedFormatted, _ := json.MarshalIndent(expectedData, "", "  ")
			actualFormatted, _ := json.MarshalIndent(actualData, "", "  ")

			if string(expectedFormatted) != string(actualFormatted) {
				t.Errorf("Expected JSON:\n%s\nGot JSON:\n%s", string(expectedFormatted), string(actualFormatted))
			}
		})
	}
}
