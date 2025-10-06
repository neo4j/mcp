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

func TestReadCypherHandler(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	t.Run("successful cypher execution with parameters", func(t *testing.T) {
		mockDB := mocks.NewMockDatabaseService(ctrl)
		mockDB.EXPECT().
			ExecuteReadQuery(gomock.Any(), "MATCH (n:Person {name: $name}) RETURN n", map[string]any{"name": "Alice"}, "testdb").
			Return([]*neo4j.Record{}, nil)
		mockDB.EXPECT().
			GetQueryType(gomock.Any(), "MATCH (n:Person {name: $name}) RETURN n", map[string]any{"name": "Alice"}, "testdb").
			Return(neo4j.StatementTypeReadOnly, nil)
		mockDB.EXPECT().
			Neo4jRecordsToJSON(gomock.Any()).
			Return(`[{"n": {"name": "Alice"}}]`, nil)

		deps := &ToolDependencies{
			Config:    &config.Config{Database: "testdb"},
			DBService: mockDB,
		}

		handler := ReadCypherHandler(deps)
		request := mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Arguments: map[string]any{
					"query":  "MATCH (n:Person {name: $name}) RETURN n",
					"params": map[string]any{"name": "Alice"},
				},
			},
		}

		result, err := handler(context.Background(), request)

		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
		if result == nil || result.IsError {
			t.Error("Expected success result")
		}
	})

	t.Run("successful cypher execution without parameters", func(t *testing.T) {
		mockDB := mocks.NewMockDatabaseService(ctrl)
		mockDB.EXPECT().
			GetQueryType(gomock.Any(), "MATCH (n) RETURN count(n)", gomock.Nil(), "testdb").
			Return(neo4j.StatementTypeReadOnly, nil)
		mockDB.EXPECT().
			ExecuteReadQuery(gomock.Any(), "MATCH (n) RETURN count(n)", gomock.Nil(), "testdb").
			Return([]*neo4j.Record{}, nil)
		mockDB.EXPECT().
			Neo4jRecordsToJSON(gomock.Any()).
			Return(`[{"count(n)": 42}]`, nil)

		deps := &ToolDependencies{
			Config:    &config.Config{Database: "testdb"},
			DBService: mockDB,
		}

		handler := ReadCypherHandler(deps)
		request := mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Arguments: map[string]any{
					"query": "MATCH (n) RETURN count(n)",
				},
			},
		}

		result, err := handler(context.Background(), request)

		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
		if result == nil || result.IsError {
			t.Error("Expected success result")
		}
	})

	t.Run("invalid arguments binding", func(t *testing.T) {
		mockDB := mocks.NewMockDatabaseService(ctrl)

		deps := &ToolDependencies{
			Config:    &config.Config{Database: "testdb"},
			DBService: mockDB,
		}

		handler := ReadCypherHandler(deps)
		// Test with invalid argument structure that should cause BindArguments to fail
		request := mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Arguments: "invalid string instead of map",
			},
		}

		result, err := handler(context.Background(), request)

		if err != nil {
			t.Errorf("Expected no error from handler, got: %v", err)
		}
		if result == nil || !result.IsError {
			t.Error("Expected error result for invalid arguments")
		}
	})

	t.Run("missing required arguments", func(t *testing.T) {
		mockDB := mocks.NewMockDatabaseService(ctrl)
		// The handler should NOT call ExecuteReadQuery when query is empty
		// No expectations set for mockDB since it shouldn't be called

		deps := &ToolDependencies{
			Config:    &config.Config{Database: "testdb"},
			DBService: mockDB,
		}

		handler := ReadCypherHandler(deps)
		request := mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Arguments: map[string]any{
					"invalid_field": "value",
				},
			},
		}

		result, err := handler(context.Background(), request)

		if err != nil {
			t.Errorf("Expected no error from handler, got: %v", err)
		}
		// Now the handler should return an error for empty query
		if result == nil || !result.IsError {
			t.Error("Expected error result for missing query parameter")
		}
	})

	t.Run("empty query parameter", func(t *testing.T) {
		mockDB := mocks.NewMockDatabaseService(ctrl)
		// The handler should NOT call ExecuteReadQuery when query is empty
		// No expectations set for mockDB since it shouldn't be called

		deps := &ToolDependencies{
			Config:    &config.Config{Database: "testdb"},
			DBService: mockDB,
		}

		handler := ReadCypherHandler(deps)
		request := mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Arguments: map[string]any{
					"query": "",
				},
			},
		}

		result, err := handler(context.Background(), request)

		if err != nil {
			t.Errorf("Expected no error from handler, got: %v", err)
		}
		// Handler should return an error for empty query
		if result == nil || !result.IsError {
			t.Error("Expected error result for empty query parameter")
		}
	})

	t.Run("nil database service", func(t *testing.T) {
		deps := &ToolDependencies{
			Config:    &config.Config{Database: "testdb"},
			DBService: nil,
		}

		handler := ReadCypherHandler(deps)
		request := mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Arguments: map[string]any{
					"query": "MATCH (n) RETURN n",
				},
			},
		}

		result, err := handler(context.Background(), request)

		if err != nil {
			t.Errorf("Expected no error from handler, got: %v", err)
		}
		if result == nil || !result.IsError {
			t.Error("Expected error result for nil database service")
		}
	})

	t.Run("database query execution failure", func(t *testing.T) {
		mockDB := mocks.NewMockDatabaseService(ctrl)
		mockDB.EXPECT().
			GetQueryType(gomock.Any(), "INVALID CYPHER", gomock.Nil(), "testdb").
			Return(neo4j.StatementTypeReadOnly, nil)
		mockDB.EXPECT().
			ExecuteReadQuery(gomock.Any(), "INVALID CYPHER", gomock.Nil(), "testdb").
			Return(nil, errors.New("syntax error"))

		deps := &ToolDependencies{
			Config:    &config.Config{Database: "testdb"},
			DBService: mockDB,
		}

		handler := ReadCypherHandler(deps)
		request := mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Arguments: map[string]any{
					"query": "INVALID CYPHER",
				},
			},
		}

		result, err := handler(context.Background(), request)

		if err != nil {
			t.Errorf("Expected no error from handler, got: %v", err)
		}
		if result == nil || !result.IsError {
			t.Error("Expected error result for query execution failure")
		}
	})

	t.Run("JSON formatting failure", func(t *testing.T) {
		mockDB := mocks.NewMockDatabaseService(ctrl)
		mockDB.EXPECT().
			GetQueryType(gomock.Any(), "MATCH (n) RETURN n", gomock.Nil(), "testdb").
			Return(neo4j.StatementTypeReadOnly, nil)
		mockDB.EXPECT().
			ExecuteReadQuery(gomock.Any(), "MATCH (n) RETURN n", gomock.Nil(), "testdb").
			Return([]*neo4j.Record{}, nil)
		mockDB.EXPECT().
			Neo4jRecordsToJSON(gomock.Any()).
			Return("", errors.New("JSON marshaling failed"))

		deps := &ToolDependencies{
			Config:    &config.Config{Database: "testdb"},
			DBService: mockDB,
		}

		handler := ReadCypherHandler(deps)
		request := mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Arguments: map[string]any{
					"query": "MATCH (n) RETURN n",
				},
			},
		}

		result, err := handler(context.Background(), request)

		if err != nil {
			t.Errorf("Expected no error from handler, got: %v", err)
		}
		if result == nil || !result.IsError {
			t.Error("Expected error result for JSON formatting failure")
		}
	})

	t.Run("non-read query type returns error", func(t *testing.T) {
		mockDB := mocks.NewMockDatabaseService(ctrl)
		mockDB.EXPECT().
			GetQueryType(gomock.Any(), "CREATE (n:Test)", gomock.Nil(), "testdb").
			Return(neo4j.StatementTypeWriteOnly, nil)

		deps := &ToolDependencies{
			Config:    &config.Config{Database: "testdb"},
			DBService: mockDB,
		}

		handler := ReadCypherHandler(deps)
		request := mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Arguments: map[string]any{
					"query": "CREATE (n:Test)",
				},
			},
		}

		result, err := handler(context.Background(), request)
		if err != nil {
			t.Errorf("Expected no error from handler, got: %v", err)
		}
		if result == nil || !result.IsError {
			t.Error("Expected error result for non-read query type")
		}
	})

	t.Run("explain query failure", func(t *testing.T) {
		mockDB := mocks.NewMockDatabaseService(ctrl)
		mockDB.EXPECT().
			GetQueryType(gomock.Any(), "MATCH (n) RETURN n", gomock.Nil(), "testdb").
			Return(neo4j.StatementTypeUnknown, errors.New("driver error"))

		deps := &ToolDependencies{
			Config:    &config.Config{Database: "testdb"},
			DBService: mockDB,
		}

		handler := ReadCypherHandler(deps)
		request := mcp.CallToolRequest{
			Params: mcp.CallToolParams{
				Arguments: map[string]any{
					"query": "MATCH (n) RETURN n",
				},
			},
		}

		result, err := handler(context.Background(), request)
		if err != nil {
			t.Errorf("Expected no error from handler, got: %v", err)
		}
		if result == nil || !result.IsError {
			t.Error("Expected error result for explain failure")
		}
	})
}
