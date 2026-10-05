// Copyright (c) "Neo4j"
// Neo4j Sweden AB [http://neo4j.com]

package cypher_test

import (
	"context"
	"errors"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	analytics "github.com/neo4j/mcp/internal/analytics/mocks"
	db "github.com/neo4j/mcp/internal/database/mocks"
	"github.com/neo4j/mcp/internal/tools"
	"github.com/neo4j/mcp/internal/tools/cypher"
	"github.com/neo4j/neo4j-go-driver/v6/neo4j"
	"go.uber.org/mock/gomock"
)

func TestReadCypherHandler(t *testing.T) {
	ctrl := gomock.NewController(t)
	analyticsService := analytics.NewMockService(ctrl)
	// Note: Handlers no longer emit events directly - events are emitted via hooks in server.go
	defer ctrl.Finish()

	t.Run("successful cypher execution with parameters", func(t *testing.T) {
		mockDB := db.NewMockService(ctrl)
		mockDB.EXPECT().
			ExecuteReadQuery(gomock.Any(), "MATCH (n:Person {name: $name}) RETURN n", map[string]any{"name": "Alice"}).
			Return([]*neo4j.Record{}, nil)
		mockDB.EXPECT().
			GetQueryType(gomock.Any(), "MATCH (n:Person {name: $name}) RETURN n", map[string]any{"name": "Alice"}).
			Return(neo4j.QueryTypeReadOnly, nil)
		mockDB.EXPECT().
			Neo4jRecordsToJSON(gomock.Any()).
			Return(`[{"n": {"name": "Alice"}}]`, nil)

		deps := &tools.ToolDependencies{
			DBService:        mockDB,
			AnalyticsService: analyticsService,
		}

		handler := cypher.ReadCypherHandler(deps)
		input := cypher.ReadCypherInput{
			Query:  "MATCH (n:Person {name: $name}) RETURN n",
			Params: cypher.Params{"name": "Alice"},
		}

		result, _, err := handler(context.Background(), &mcp.CallToolRequest{}, input)

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
			GetQueryType(gomock.Any(), "MATCH (n) RETURN count(n)", gomock.Nil()).
			Return(neo4j.QueryTypeReadOnly, nil)
		mockDB.EXPECT().
			ExecuteReadQuery(gomock.Any(), "MATCH (n) RETURN count(n)", gomock.Nil()).
			Return([]*neo4j.Record{}, nil)
		mockDB.EXPECT().
			Neo4jRecordsToJSON(gomock.Any()).
			Return(`[{"count(n)": 42}]`, nil)

		deps := &tools.ToolDependencies{
			DBService:        mockDB,
			AnalyticsService: analyticsService,
		}

		handler := cypher.ReadCypherHandler(deps)
		input := cypher.ReadCypherInput{
			Query: "MATCH (n) RETURN count(n)",
		}

		result, _, err := handler(context.Background(), &mcp.CallToolRequest{}, input)

		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
		if result == nil || result.IsError {
			t.Error("Expected success result")
		}
	})

	t.Run("missing required arguments", func(t *testing.T) {
		mockDB := db.NewMockService(ctrl)
		// The handler should NOT call ExecuteReadQuery when query is empty
		// No expectations set for mockDB since it shouldn't be called

		deps := &tools.ToolDependencies{
			DBService:        mockDB,
			AnalyticsService: analyticsService,
		}

		handler := cypher.ReadCypherHandler(deps)
		input := cypher.ReadCypherInput{}

		result, _, err := handler(context.Background(), &mcp.CallToolRequest{}, input)

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
		// The handler should NOT call ExecuteReadQuery when query is empty
		// No expectations set for mockDB since it shouldn't be called

		deps := &tools.ToolDependencies{
			DBService:        mockDB,
			AnalyticsService: analyticsService,
		}

		handler := cypher.ReadCypherHandler(deps)
		input := cypher.ReadCypherInput{
			Query: "",
		}

		result, _, err := handler(context.Background(), &mcp.CallToolRequest{}, input)

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

		handler := cypher.ReadCypherHandler(deps)
		input := cypher.ReadCypherInput{
			Query: "MATCH (n) RETURN n",
		}

		result, _, err := handler(context.Background(), &mcp.CallToolRequest{}, input)

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

		handler := cypher.ReadCypherHandler(deps)
		result, _, err := handler(context.Background(), &mcp.CallToolRequest{}, cypher.ReadCypherInput{})

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
			GetQueryType(gomock.Any(), "INVALID CYPHER", gomock.Nil()).
			Return(neo4j.QueryTypeReadOnly, nil)
		mockDB.EXPECT().
			ExecuteReadQuery(gomock.Any(), "INVALID CYPHER", gomock.Nil()).
			Return(nil, errors.New("syntax error"))

		deps := &tools.ToolDependencies{
			DBService:        mockDB,
			AnalyticsService: analyticsService,
		}

		handler := cypher.ReadCypherHandler(deps)
		input := cypher.ReadCypherInput{
			Query: "INVALID CYPHER",
		}

		result, _, err := handler(context.Background(), &mcp.CallToolRequest{}, input)

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
			GetQueryType(gomock.Any(), "MATCH (n) RETURN n", gomock.Nil()).
			Return(neo4j.QueryTypeReadOnly, nil)
		mockDB.EXPECT().
			ExecuteReadQuery(gomock.Any(), "MATCH (n) RETURN n", gomock.Nil()).
			Return([]*neo4j.Record{}, nil)
		mockDB.EXPECT().
			Neo4jRecordsToJSON(gomock.Any()).
			Return("", errors.New("JSON marshaling failed"))

		deps := &tools.ToolDependencies{
			DBService:        mockDB,
			AnalyticsService: analyticsService,
		}

		handler := cypher.ReadCypherHandler(deps)
		input := cypher.ReadCypherInput{
			Query: "MATCH (n) RETURN n",
		}

		result, _, err := handler(context.Background(), &mcp.CallToolRequest{}, input)

		if err != nil {
			t.Errorf("Expected no error from handler, got: %v", err)
		}
		if result == nil || !result.IsError {
			t.Error("Expected error result for JSON formatting failure")
		}
	})

	t.Run("non-read query type returns error", func(t *testing.T) {
		mockDB := db.NewMockService(ctrl)
		mockDB.EXPECT().
			GetQueryType(gomock.Any(), "CREATE (n:Test)", gomock.Nil()).
			Return(neo4j.QueryTypeWriteOnly, nil)

		deps := &tools.ToolDependencies{
			DBService:        mockDB,
			AnalyticsService: analyticsService,
		}

		handler := cypher.ReadCypherHandler(deps)
		input := cypher.ReadCypherInput{
			Query: "CREATE (n:Test)",
		}

		result, _, err := handler(context.Background(), &mcp.CallToolRequest{}, input)
		if err != nil {
			t.Errorf("Expected no error from handler, got: %v", err)
		}
		if result == nil || !result.IsError {
			t.Error("Expected error result for non-read query type")
		}
	})

	t.Run("explain query failure", func(t *testing.T) {
		mockDB := db.NewMockService(ctrl)
		mockDB.EXPECT().
			GetQueryType(gomock.Any(), "MATCH (n) RETURN n", gomock.Nil()).
			Return(neo4j.QueryTypeUnknown, errors.New("driver error"))

		deps := &tools.ToolDependencies{
			DBService:        mockDB,
			AnalyticsService: analyticsService,
		}

		handler := cypher.ReadCypherHandler(deps)
		input := cypher.ReadCypherInput{
			Query: "MATCH (n) RETURN n",
		}

		result, _, err := handler(context.Background(), &mcp.CallToolRequest{}, input)
		if err != nil {
			t.Errorf("Expected no error from handler, got: %v", err)
		}
		if result == nil || !result.IsError {
			t.Error("Expected error result for explain failure")
		}
	})
}
