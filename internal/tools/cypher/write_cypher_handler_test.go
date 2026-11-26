// Copyright (c) "Neo4j"
// Neo4j Sweden AB [http://neo4j.com]

package cypher_test

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
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
		input := cypher.WriteCypherInput{
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
		input := cypher.WriteCypherInput{
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
		// The handler should NOT call ExecuteWriteQuery when query is empty
		// No expectations set for mockDB since it shouldn't be called

		deps := &tools.ToolDependencies{
			DBService:        mockDB,
			AnalyticsService: analyticsService,
		}

		handler := cypher.WriteCypherHandler(deps)
		input := cypher.WriteCypherInput{}

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
		// The handler should NOT call ExecuteWriteQuery when query is empty
		// No expectations set for mockDB since it shouldn't be called

		deps := &tools.ToolDependencies{
			DBService:        mockDB,
			AnalyticsService: analyticsService,
		}

		handler := cypher.WriteCypherHandler(deps)
		input := cypher.WriteCypherInput{
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

		handler := cypher.WriteCypherHandler(deps)
		input := cypher.WriteCypherInput{
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

		handler := cypher.WriteCypherHandler(deps)
		result, _, err := handler(context.Background(), &mcp.CallToolRequest{}, cypher.WriteCypherInput{})

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
		input := cypher.WriteCypherInput{
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
		input := cypher.WriteCypherInput{
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
}

func TestWriteCypherHandler_OmitEmbeddings(t *testing.T) {
	const query = "MATCH (m:Movie) SET m.seen = true RETURN m.embedding AS e"
	embedding := make([]any, 1536)
	for i := range embedding {
		embedding[i] = 0.25
	}

	tests := []struct {
		name           string
		omitEmbeddings bool
		want           any
	}{
		{name: "default leaves results unchanged", omitEmbeddings: false, want: embedding},
		{name: "omitEmbeddings replaces embeddings", omitEmbeddings: true, want: "<embedding: 1536 values omitted>"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			records := []*neo4j.Record{{Keys: []string{"e"}, Values: []any{embedding}}}

			mockDB := db.NewMockService(ctrl)
			mockDB.EXPECT().ExecuteWriteQuery(gomock.Any(), query, gomock.Nil()).Return(records, nil)
			var formatted []*neo4j.Record
			mockDB.EXPECT().Neo4jRecordsToJSON(gomock.Any()).DoAndReturn(func(r []*neo4j.Record) (string, error) {
				formatted = r
				return "[]", nil
			})

			handler := cypher.WriteCypherHandler(&tools.ToolDependencies{DBService: mockDB, AnalyticsService: analytics.NewMockService(ctrl)})
			result, _, err := handler(context.Background(), &mcp.CallToolRequest{}, cypher.WriteCypherInput{Query: query, OmitEmbeddings: tt.omitEmbeddings})

			if err != nil || result == nil || result.IsError {
				t.Fatalf("expected success, got result=%+v err=%v", result, err)
			}
			if !tt.omitEmbeddings && formatted[0] != records[0] {
				t.Error("expected the driver records to be formatted as-is when omitEmbeddings is false")
			}
			if !reflect.DeepEqual(formatted[0].Values[0], tt.want) {
				t.Errorf("formatted value = %v, want %v", formatted[0].Values[0], tt.want)
			}
		})
	}
}
