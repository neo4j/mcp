//
// Copyright (c) "Neo4j"
// Neo4j Sweden AB [http://neo4j.com]

package cypher_test

import (
	"context"
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

func TestWriteCypherHandler(t *testing.T) {
	ctrl := gomock.NewController(t)
	analyticsService := analytics.NewMockService(ctrl)
	// Note: Handlers no longer emit events directly - events are emitted via hooks in server.go
	defer ctrl.Finish()

	t.Run("successful cypher execution with parameters", func(t *testing.T) {
		mockDB := db.NewMockService(ctrl)
		mockDB.EXPECT().
			ExecuteWriteQuery(gomock.Any(), "MATCH (n:Person {name: $name}) RETURN n", map[string]any{"name": "Alice"}).
			Return([]*neo4j.Record{}, nil)
		mockDB.EXPECT().
			Neo4jRecordsToJSON(gomock.Any()).
			Return(`[{"n": {"name": "Alice"}}]`, nil)

		deps := &tools.ToolDependencies{
			DBService:        mockDB,
			AnalyticsService: analyticsService,
		}

		handler := cypher.WriteCypherHandler(deps)
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
		mockDB := db.NewMockService(ctrl)
		mockDB.EXPECT().
			ExecuteWriteQuery(gomock.Any(), "MATCH (n) RETURN count(n)", gomock.Nil()).
			Return([]*neo4j.Record{}, nil)
		mockDB.EXPECT().
			Neo4jRecordsToJSON(gomock.Any()).
			Return(`[{"count(n)": 42}]`, nil)

		deps := &tools.ToolDependencies{
			DBService:        mockDB,
			AnalyticsService: analyticsService,
		}

		handler := cypher.WriteCypherHandler(deps)
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
		mockDB := db.NewMockService(ctrl)

		deps := &tools.ToolDependencies{
			DBService:        mockDB,
			AnalyticsService: analyticsService,
		}

		handler := cypher.WriteCypherHandler(deps)
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
		mockDB := db.NewMockService(ctrl)
		// The handler should NOT call ExecuteWriteQuery when query is empty
		// No expectations set for mockDB since it shouldn't be called

		deps := &tools.ToolDependencies{
			DBService:        mockDB,
			AnalyticsService: analyticsService,
		}

		handler := cypher.WriteCypherHandler(deps)
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
		mockDB := db.NewMockService(ctrl)
		// The handler should NOT call ExecuteWriteQuery when query is empty
		// No expectations set for mockDB since it shouldn't be called

		deps := &tools.ToolDependencies{
			DBService:        mockDB,
			AnalyticsService: analyticsService,
		}

		handler := cypher.WriteCypherHandler(deps)
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
		deps := &tools.ToolDependencies{
			DBService:        nil,
			AnalyticsService: analyticsService,
		}

		handler := cypher.WriteCypherHandler(deps)
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
	t.Run("nil analytics service", func(t *testing.T) {
		mockDB := db.NewMockService(ctrl)
		deps := &tools.ToolDependencies{
			DBService:        mockDB,
			AnalyticsService: nil,
		}

		handler := cypher.WriteCypherHandler(deps)
		result, err := handler(context.Background(), mcp.CallToolRequest{})

		if err != nil {
			t.Errorf("Expected no error from handler, got: %v", err)
		}
		if result == nil || !result.IsError {
			t.Error("Expected error result for nil analytics service")
		}
	})

	t.Run("database query execution failure", func(t *testing.T) {
		mockDB := db.NewMockService(ctrl)
		mockDB.EXPECT().
			ExecuteWriteQuery(gomock.Any(), "INVALID CYPHER", gomock.Nil()).
			Return(nil, errors.New("syntax error"))

		deps := &tools.ToolDependencies{
			DBService:        mockDB,
			AnalyticsService: analyticsService,
		}

		handler := cypher.WriteCypherHandler(deps)
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
		mockDB := db.NewMockService(ctrl)
		mockDB.EXPECT().
			ExecuteWriteQuery(gomock.Any(), "MATCH (n) RETURN n", gomock.Nil()).
			Return([]*neo4j.Record{}, nil)
		mockDB.EXPECT().
			Neo4jRecordsToJSON(gomock.Any()).
			Return("", errors.New("JSON marshaling failed"))

		deps := &tools.ToolDependencies{
			DBService:        mockDB,
			AnalyticsService: analyticsService,
		}

		handler := cypher.WriteCypherHandler(deps)
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
}
