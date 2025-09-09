package tools

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/neo4j/mcp/internal/config"
	"github.com/neo4j/mcp/internal/database/mocks"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"go.uber.org/mock/gomock"
)

func TestGetSchemaHandler(t *testing.T) {
	tests := []struct {
		name        string
		setupMock   func(*mocks.MockDatabaseService)
		expectError bool
	}{
		{
			name: "successful schema retrieval",
			setupMock: func(mockDB *mocks.MockDatabaseService) {
				// Expect ExecuteReadQuery to be called once with the schema query
				mockDB.EXPECT().
					ExecuteReadQuery(gomock.Any(), gomock.Any(), gomock.Nil(), "testdb").
					DoAndReturn(func(ctx context.Context, cypher string, params map[string]any, database string) ([]*neo4j.Record, error) {
						// Verify the correct query is being used
						if !strings.Contains(cypher, "CALL apoc.meta.schema()") {
							t.Errorf("Expected APOC schema query, got: %s", cypher)
						}
						// Return empty records (we can't easily create real neo4j.Record instances)
						return []*neo4j.Record{}, nil
					}).
					Times(1)

				// Expect Neo4jRecordsToJSON to be called once
				mockDB.EXPECT().
					Neo4jRecordsToJSON(gomock.Any()).
					Return(`[{"key": "test", "value": {"type": "node"}}]`, nil).
					Times(1)
			},
			expectError: false,
		},
		{
			name: "database query failure",
			setupMock: func(mockDB *mocks.MockDatabaseService) {
				// Mock ExecuteReadQuery to return an error
				mockDB.EXPECT().
					ExecuteReadQuery(gomock.Any(), gomock.Any(), gomock.Nil(), "testdb").
					Return(nil, errors.New("connection failed")).
					Times(1)

				// Neo4jRecordsToJSON should not be called when query fails
				mockDB.EXPECT().
					Neo4jRecordsToJSON(gomock.Any()).
					Times(0)
			},
			expectError: true,
		},
		{
			name: "JSON formatting failure",
			setupMock: func(mockDB *mocks.MockDatabaseService) {
				// Mock successful query execution
				mockDB.EXPECT().
					ExecuteReadQuery(gomock.Any(), gomock.Any(), gomock.Nil(), "testdb").
					Return([]*neo4j.Record{}, nil).
					Times(1)

				// Mock Neo4jRecordsToJSON to return an error
				mockDB.EXPECT().
					Neo4jRecordsToJSON(gomock.Any()).
					Return("", errors.New("JSON marshaling failed")).
					Times(1)
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a new mock controller for each test
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			// Create the mock database service
			mockDB := mocks.NewMockDatabaseService(ctrl)

			// Setup mock expectations
			tt.setupMock(mockDB)

			// Create dependencies with the mock
			deps := &ToolDependencies{
				Driver:    nil, // Not needed with interface
				Config:    &config.Config{Database: "testdb"},
				DBService: mockDB,
			}

			// Get the handler and call it
			handler := GetSchemaHandler(deps)
			result, err := handler(context.Background(), mcp.CallToolRequest{})

			// Verify no error from handler function itself
			if err != nil {
				t.Errorf("Expected no error from handler, got: %v", err)
			}

			// Check if we got expected error result
			if tt.expectError {
				if result == nil || !result.IsError {
					t.Errorf("Expected error result")
				}
			} else {
				if result == nil || result.IsError {
					t.Errorf("Expected success result")
				}
			}
		})
	}
}

func TestHandleGetSchema(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	t.Run("successful execution with specific assertions", func(t *testing.T) {
		mockDB := mocks.NewMockDatabaseService(ctrl)
		config := &config.Config{Database: "neo4j"}

		// Set up precise expectations
		mockDB.EXPECT().
			ExecuteReadQuery(
				gomock.Any(), // context
				schemaQuery,  // exact query constant
				gomock.Nil(), // nil params
				"neo4j",      // exact database name
			).
			Return([]*neo4j.Record{}, nil).
			Times(1)

		mockDB.EXPECT().
			Neo4jRecordsToJSON(gomock.Eq([]*neo4j.Record{})).
			Return(`{"schema": "data"}`, nil).
			Times(1)

		result, err := handleGetSchema(context.Background(), mockDB, config)

		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}

		if result == nil {
			t.Error("Expected result, got nil")
			return
		}

		if result.IsError {
			t.Error("Expected success result, got error")
		}
	})
}

func TestGetSchemaHandler_InputValidation(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	t.Run("nil database service", func(t *testing.T) {
		deps := &ToolDependencies{
			Driver:    nil,
			Config:    &config.Config{Database: "test"},
			DBService: nil,
		}

		handler := GetSchemaHandler(deps)
		result, err := handler(context.Background(), mcp.CallToolRequest{})

		if err != nil {
			t.Errorf("Expected no error from handler, got: %v", err)
		}

		if result == nil || !result.IsError {
			t.Error("Expected error result for nil database service")
		}
	})

	t.Run("nil config", func(t *testing.T) {
		// Create a mock but don't set any expectations since it shouldn't be called
		mockDB := mocks.NewMockDatabaseService(ctrl)

		deps := &ToolDependencies{
			Driver:    nil,
			Config:    nil,
			DBService: mockDB,
		}

		handler := GetSchemaHandler(deps)
		result, err := handler(context.Background(), mcp.CallToolRequest{})

		if err != nil {
			t.Errorf("Expected no error from handler, got: %v", err)
		}

		if result == nil || !result.IsError {
			t.Error("Expected error result for nil config")
		}
	})
}
