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

func TestGetSchemaHandler(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	t.Run("successful schema retrieval", func(t *testing.T) {
		mockDB := mocks.NewMockService(ctrl)
		mockDB.EXPECT().
			ExecuteReadQuery(gomock.Any(), gomock.Any(), gomock.Nil(), "testdb").
			Return([]*neo4j.Record{
				{
					Values: []any{"value1"},
					Keys:   []string{"key1"},
				},
			}, nil)
		mockDB.EXPECT().
			Neo4jRecordsToJSON(gomock.Any()).
			Return(`{"schema": "data"}`, nil)

		deps := &ToolDependencies{
			Config:    &config.Config{Database: "testdb"},
			DBService: mockDB,
		}

		handler := GetSchemaHandler(deps)
		result, err := handler(context.Background(), mcp.CallToolRequest{})

		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
		if result == nil || result.IsError {
			t.Error("Expected success result")
		}
	})

	t.Run("database query failure", func(t *testing.T) {
		mockDB := mocks.NewMockService(ctrl)
		mockDB.EXPECT().
			ExecuteReadQuery(gomock.Any(), gomock.Any(), gomock.Nil(), "testdb").
			Return(nil, errors.New("connection failed"))

		deps := &ToolDependencies{
			Config:    &config.Config{Database: "testdb"},
			DBService: mockDB,
		}

		handler := GetSchemaHandler(deps)
		result, err := handler(context.Background(), mcp.CallToolRequest{})

		if err != nil {
			t.Errorf("Expected no error from handler, got: %v", err)
		}
		if result == nil || !result.IsError {
			t.Error("Expected error result")
		}
	})

	t.Run("JSON formatting failure", func(t *testing.T) {
		mockDB := mocks.NewMockService(ctrl)
		mockDB.EXPECT().
			ExecuteReadQuery(gomock.Any(), gomock.Any(), gomock.Nil(), "testdb").
			Return([]*neo4j.Record{
				{
					Values: []any{"value1"},
					Keys:   []string{"key1"},
				},
			}, nil)
		mockDB.EXPECT().
			Neo4jRecordsToJSON(gomock.Any()).
			Return("", errors.New("JSON marshaling failed"))

		deps := &ToolDependencies{
			Config:    &config.Config{Database: "testdb"},
			DBService: mockDB,
		}

		handler := GetSchemaHandler(deps)
		result, err := handler(context.Background(), mcp.CallToolRequest{})

		if err != nil {
			t.Errorf("Expected no error from handler, got: %v", err)
		}
		if result == nil || !result.IsError {
			t.Error("Expected error result")
		}
	})

	t.Run("nil database service", func(t *testing.T) {
		deps := &ToolDependencies{
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
	t.Run("No records returned from apoc query (empty database)", func(t *testing.T) {
		mockDB := mocks.NewMockService(ctrl)
		mockDB.EXPECT().
			ExecuteReadQuery(gomock.Any(), gomock.Any(), gomock.Nil(), "testdb").
			Return([]*neo4j.Record{}, nil)

		deps := &ToolDependencies{
			Config:    &config.Config{Database: "testdb"},
			DBService: mockDB,
		}

		handler := GetSchemaHandler(deps)
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
