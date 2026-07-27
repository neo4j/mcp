// Copyright (c) "Neo4j"
// Neo4j Sweden AB [http://neo4j.com]

package vector_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/neo4j/mcp/internal/config"
	db "github.com/neo4j/mcp/internal/database/mocks"
	"github.com/neo4j/mcp/internal/tools"
	"github.com/neo4j/mcp/internal/tools/vector"
	"github.com/neo4j/neo4j-go-driver/v6/neo4j"
	"go.uber.org/mock/gomock"
)

// staticUseSearch returns a useSearchClause function that always yields b.
func staticUseSearch(b bool) func() bool {
	return func() bool { return b }
}

// staticUseAiTextEmbed returns a useAiTextEmbed function that always yields b.
func staticUseAiTextEmbed(b bool) func() bool {
	return func() bool { return b }
}

// embedCfg returns a minimal, valid OpenAI embedding config.
func embedCfg() config.EmbeddingConfig {
	return config.EmbeddingConfig{
		Provider: config.EmbeddingProviderOpenAI,
		APIKey:   "sk-placeholder",
		Model:    "text-embedding-3-small",
	}
}

// buildRequest builds a CallToolRequest with the given arguments map.
func buildRequest(t *testing.T, args map[string]any) mcp.CallToolRequest {
	t.Helper()
	return mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: args,
		},
	}
}

// singleIndexRecord returns a single-element record slice for SHOW VECTOR INDEXES.
func singleIndexRecord() []*neo4j.Record {
	return []*neo4j.Record{
		{
			Keys: []string{"name", "entityType", "labelsOrTypes", "properties", "options"},
			Values: []any{
				"moviePlots",
				"NODE",
				[]any{"Movie"},
				[]any{"embedding"},
				map[string]any{},
			},
		},
	}
}

func TestVectorSearchHandler_NilDBService(t *testing.T) {
	t.Parallel()
	deps := &tools.ToolDependencies{DBService: nil}
	handler := vector.SearchHandler(deps, embedCfg(), staticUseSearch(true), staticUseAiTextEmbed(true))
	result, err := handler(context.Background(), buildRequest(t, map[string]any{"query": "hello"}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected error result for nil DBService")
	}
}

func TestVectorSearchHandler_EmbeddingNotConfigured(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockDB := db.NewMockService(ctrl)
	// No expectations — handler should return early before any DB call
	deps := &tools.ToolDependencies{DBService: mockDB}
	emb := config.EmbeddingConfig{} // empty provider
	handler := vector.SearchHandler(deps, emb, staticUseSearch(true), staticUseAiTextEmbed(true))
	result, err := handler(context.Background(), buildRequest(t, map[string]any{"query": "hello"}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected error result when embedding not configured")
	}
}

func TestVectorSearchHandler_EmptyQuery(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockDB := db.NewMockService(ctrl)
	// No DB calls expected
	deps := &tools.ToolDependencies{DBService: mockDB}
	handler := vector.SearchHandler(deps, embedCfg(), staticUseSearch(true), staticUseAiTextEmbed(true))
	result, err := handler(context.Background(), buildRequest(t, map[string]any{"query": ""}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected error result for empty query")
	}
}

func TestVectorSearchHandler_TopKDefault(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockDB := db.NewMockService(ctrl)

	// Expect index resolution then query execution
	mockDB.EXPECT().
		ExecuteReadQuery(gomock.Any(), gomock.Any(), gomock.Nil()).
		Return(singleIndexRecord(), nil)
	mockDB.EXPECT().
		ExecuteReadQuery(gomock.Any(), gomock.Any(), gomock.AssignableToTypeOf(map[string]any{})).
		DoAndReturn(func(_ context.Context, _ string, params map[string]any) ([]*neo4j.Record, error) {
			topK, ok := params["topK"]
			if !ok {
				t.Error("topK param missing")
			} else if topK != 5 {
				t.Errorf("expected default topK=5, got %v", topK)
			}
			return []*neo4j.Record{}, nil
		})
	mockDB.EXPECT().
		Neo4jRecordsToJSON(gomock.Any()).
		Return("[]", nil)

	deps := &tools.ToolDependencies{DBService: mockDB}
	handler := vector.SearchHandler(deps, embedCfg(), staticUseSearch(true), staticUseAiTextEmbed(true))
	result, err := handler(context.Background(), buildRequest(t, map[string]any{"query": "sci-fi movies"}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Errorf("unexpected error result: %v", extractText(result))
	}
}

func TestVectorSearchHandler_TopKClamp(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockDB := db.NewMockService(ctrl)

	mockDB.EXPECT().
		ExecuteReadQuery(gomock.Any(), gomock.Any(), gomock.Nil()).
		Return(singleIndexRecord(), nil)
	mockDB.EXPECT().
		ExecuteReadQuery(gomock.Any(), gomock.Any(), gomock.AssignableToTypeOf(map[string]any{})).
		DoAndReturn(func(_ context.Context, _ string, params map[string]any) ([]*neo4j.Record, error) {
			if params["topK"] != 100 {
				t.Errorf("expected clamped topK=100, got %v", params["topK"])
			}
			return []*neo4j.Record{}, nil
		})
	mockDB.EXPECT().
		Neo4jRecordsToJSON(gomock.Any()).
		Return("[]", nil)

	deps := &tools.ToolDependencies{DBService: mockDB}
	handler := vector.SearchHandler(deps, embedCfg(), staticUseSearch(true), staticUseAiTextEmbed(true))
	// Request topK=500 — should be clamped to 100
	result, err := handler(context.Background(), buildRequest(t, map[string]any{"query": "action", "topK": 500}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Errorf("unexpected error result: %v", extractText(result))
	}
}

func TestVectorSearchHandler_InvalidFilterOperator(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockDB := db.NewMockService(ctrl)
	// No DB calls expected

	deps := &tools.ToolDependencies{DBService: mockDB}
	handler := vector.SearchHandler(deps, embedCfg(), staticUseSearch(true), staticUseAiTextEmbed(true))
	filters := []any{
		map[string]any{"property": "genre", "operator": "LIKE", "value": "%Horror%"},
	}
	result, err := handler(context.Background(), buildRequest(t, map[string]any{"query": "test", "filters": filters}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected error for invalid filter operator")
	}
}

func TestVectorSearchHandler_InvalidFilterProperty(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockDB := db.NewMockService(ctrl)
	// No DB calls expected

	deps := &tools.ToolDependencies{DBService: mockDB}
	handler := vector.SearchHandler(deps, embedCfg(), staticUseSearch(true), staticUseAiTextEmbed(true))
	filters := []any{
		map[string]any{"property": "bad prop!", "operator": "=", "value": "x"},
	}
	result, err := handler(context.Background(), buildRequest(t, map[string]any{"query": "test", "filters": filters}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected error for invalid filter property name")
	}
}

func TestVectorSearchHandler_ErrorSanitization(t *testing.T) {
	t.Parallel()
	// Critical security test: a driver error that might echo back the embedConfig value
	// must NOT appear in the returned tool error text.
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	sensitiveToken := "sk-super-secret-token-12345"
	emb := config.EmbeddingConfig{
		Provider: config.EmbeddingProviderOpenAI,
		APIKey:   sensitiveToken,
		Model:    "text-embedding-3-small",
	}

	mockDB := db.NewMockService(ctrl)
	mockDB.EXPECT().
		ExecuteReadQuery(gomock.Any(), gomock.Any(), gomock.Nil()).
		Return(singleIndexRecord(), nil)

	// Driver error that happens to contain the secret token (e.g., reflects back params)
	driverErr := errors.New("query execution failed: params={embedConfig: {token: " + sensitiveToken + "}}")
	mockDB.EXPECT().
		ExecuteReadQuery(gomock.Any(), gomock.Any(), gomock.AssignableToTypeOf(map[string]any{})).
		Return(nil, driverErr)

	deps := &tools.ToolDependencies{DBService: mockDB}
	handler := vector.SearchHandler(deps, emb, staticUseSearch(true), staticUseAiTextEmbed(true))
	result, err := handler(context.Background(), buildRequest(t, map[string]any{"query": "find movies"}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected error result")
	}

	// The error text returned to the client MUST NOT contain the sensitive token.
	errText := extractText(result)
	if strings.Contains(errText, sensitiveToken) {
		t.Errorf("SECURITY VIOLATION: sensitive token leaked in client error: %q", errText)
	}
}

func TestVectorSearchHandler_EmbeddingPropertyStrippedInQuery(t *testing.T) {
	t.Parallel()
	// Verify that the generated Cypher projects the embedding property as null.
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockDB := db.NewMockService(ctrl)

	mockDB.EXPECT().
		ExecuteReadQuery(gomock.Any(), gomock.Any(), gomock.Nil()).
		Return(singleIndexRecord(), nil)
	mockDB.EXPECT().
		ExecuteReadQuery(gomock.Any(), gomock.AssignableToTypeOf(""), gomock.AssignableToTypeOf(map[string]any{})).
		DoAndReturn(func(_ context.Context, cypher string, _ map[string]any) ([]*neo4j.Record, error) {
			// The Cypher must exclude the embedding property from results.
			if !strings.Contains(cypher, "`embedding`: null") {
				t.Errorf("cypher must strip embedding property; got:\n%s", cypher)
			}
			return []*neo4j.Record{}, nil
		})
	mockDB.EXPECT().
		Neo4jRecordsToJSON(gomock.Any()).
		Return("[]", nil)

	deps := &tools.ToolDependencies{DBService: mockDB}
	handler := vector.SearchHandler(deps, embedCfg(), staticUseSearch(true), staticUseAiTextEmbed(true))
	result, err := handler(context.Background(), buildRequest(t, map[string]any{"query": "drama"}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Errorf("unexpected error: %v", extractText(result))
	}
}

func TestVectorSearchHandler_QueryNodesPath(t *testing.T) {
	t.Parallel()
	// Verify queryNodes path includes $indexName param and correct CALL.
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockDB := db.NewMockService(ctrl)

	mockDB.EXPECT().
		ExecuteReadQuery(gomock.Any(), gomock.Any(), gomock.Nil()).
		Return(singleIndexRecord(), nil)
	mockDB.EXPECT().
		ExecuteReadQuery(gomock.Any(), gomock.AssignableToTypeOf(""), gomock.AssignableToTypeOf(map[string]any{})).
		DoAndReturn(func(_ context.Context, cypher string, params map[string]any) ([]*neo4j.Record, error) {
			if !strings.Contains(cypher, "db.index.vector.queryNodes") {
				t.Errorf("expected queryNodes in cypher; got:\n%s", cypher)
			}
			if params["indexName"] != "moviePlots" {
				t.Errorf("expected indexName=moviePlots, got %v", params["indexName"])
			}
			return []*neo4j.Record{}, nil
		})
	mockDB.EXPECT().
		Neo4jRecordsToJSON(gomock.Any()).
		Return("[]", nil)

	deps := &tools.ToolDependencies{DBService: mockDB}
	// useSearchClause=false → queryNodes path
	handler := vector.SearchHandler(deps, embedCfg(), staticUseSearch(false), staticUseAiTextEmbed(true))
	result, err := handler(context.Background(), buildRequest(t, map[string]any{"query": "comedy"}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Errorf("unexpected error: %v", extractText(result))
	}
}

func TestVectorSearchHandler_FiltersIncreasesFetchK(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockDB := db.NewMockService(ctrl)

	mockDB.EXPECT().
		ExecuteReadQuery(gomock.Any(), gomock.Any(), gomock.Nil()).
		Return(singleIndexRecord(), nil)
	mockDB.EXPECT().
		ExecuteReadQuery(gomock.Any(), gomock.Any(), gomock.AssignableToTypeOf(map[string]any{})).
		DoAndReturn(func(_ context.Context, _ string, params map[string]any) ([]*neo4j.Record, error) {
			topK := params["topK"].(int)
			fetchK := params["fetchK"].(int)
			if fetchK <= topK {
				t.Errorf("with filters fetchK (%d) should be > topK (%d)", fetchK, topK)
			}
			return []*neo4j.Record{}, nil
		})
	mockDB.EXPECT().
		Neo4jRecordsToJSON(gomock.Any()).
		Return("[]", nil)

	deps := &tools.ToolDependencies{DBService: mockDB}
	handler := vector.SearchHandler(deps, embedCfg(), staticUseSearch(true), staticUseAiTextEmbed(true))
	filters := []any{
		map[string]any{"property": "genre", "operator": "=", "value": "Horror"},
	}
	result, err := handler(context.Background(), buildRequest(t, map[string]any{
		"query":   "scary movies",
		"topK":    10,
		"filters": filters,
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Errorf("unexpected error: %v", extractText(result))
	}
}

func TestVectorSearchHandler_SuccessfulSearch(t *testing.T) {
	t.Parallel()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockDB := db.NewMockService(ctrl)

	mockDB.EXPECT().
		ExecuteReadQuery(gomock.Any(), gomock.Any(), gomock.Nil()).
		Return(singleIndexRecord(), nil)
	mockDB.EXPECT().
		ExecuteReadQuery(gomock.Any(), gomock.Any(), gomock.AssignableToTypeOf(map[string]any{})).
		Return([]*neo4j.Record{}, nil)
	mockDB.EXPECT().
		Neo4jRecordsToJSON(gomock.Any()).
		Return(`[{"node":{"title":"The Matrix"},"score":0.95}]`, nil)

	deps := &tools.ToolDependencies{DBService: mockDB}
	handler := vector.SearchHandler(deps, embedCfg(), staticUseSearch(true), staticUseAiTextEmbed(true))
	result, err := handler(context.Background(), buildRequest(t, map[string]any{"query": "cyberpunk"}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Errorf("unexpected error: %v", extractText(result))
	}
	text := extractText(result)
	if !strings.Contains(text, "The Matrix") {
		t.Errorf("expected result to contain 'The Matrix', got: %s", text)
	}
}

func TestVectorSearchHandler_GenaiVectorEncodeFallback(t *testing.T) {
	t.Parallel()
	// useAiTextEmbed=false must select genai.vector.encode and the PascalCase provider
	// literal (BuildGenaiVectorEncodeConfig), not ai.text.embed.
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockDB := db.NewMockService(ctrl)

	mockDB.EXPECT().
		ExecuteReadQuery(gomock.Any(), gomock.Any(), gomock.Nil()).
		Return(singleIndexRecord(), nil)
	mockDB.EXPECT().
		ExecuteReadQuery(gomock.Any(), gomock.AssignableToTypeOf(""), gomock.AssignableToTypeOf(map[string]any{})).
		DoAndReturn(func(_ context.Context, cypher string, params map[string]any) ([]*neo4j.Record, error) {
			if !strings.Contains(cypher, "genai.vector.encode") {
				t.Errorf("expected genai.vector.encode in cypher; got:\n%s", cypher)
			}
			if strings.Contains(cypher, "ai.text.embed") {
				t.Errorf("did not expect ai.text.embed in cypher; got:\n%s", cypher)
			}
			if params["provider"] != "OpenAI" {
				t.Errorf("expected PascalCase provider 'OpenAI', got %v", params["provider"])
			}
			return []*neo4j.Record{}, nil
		})
	mockDB.EXPECT().
		Neo4jRecordsToJSON(gomock.Any()).
		Return("[]", nil)

	deps := &tools.ToolDependencies{DBService: mockDB}
	// useSearchClause=false, useAiTextEmbed=false → queryNodes + genai.vector.encode.
	handler := vector.SearchHandler(deps, embedCfg(), staticUseSearch(false), staticUseAiTextEmbed(false))
	result, err := handler(context.Background(), buildRequest(t, map[string]any{"query": "old aura instance"}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsError {
		t.Errorf("unexpected error: %v", extractText(result))
	}
}

func TestVectorSearchHandler_GenaiVectorEncodeFallback_ErrorSanitization(t *testing.T) {
	t.Parallel()
	// Same security guarantee as TestVectorSearchHandler_ErrorSanitization, but for the
	// genai.vector.encode fallback path: the API key must never leak into the client error.
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	sensitiveToken := "sk-super-secret-token-67890"
	emb := config.EmbeddingConfig{
		Provider: config.EmbeddingProviderOpenAI,
		APIKey:   sensitiveToken,
		Model:    "text-embedding-3-small",
	}

	mockDB := db.NewMockService(ctrl)
	mockDB.EXPECT().
		ExecuteReadQuery(gomock.Any(), gomock.Any(), gomock.Nil()).
		Return(singleIndexRecord(), nil)

	driverErr := errors.New("query execution failed: params={embedConfig: {token: " + sensitiveToken + "}}")
	mockDB.EXPECT().
		ExecuteReadQuery(gomock.Any(), gomock.Any(), gomock.AssignableToTypeOf(map[string]any{})).
		Return(nil, driverErr)

	deps := &tools.ToolDependencies{DBService: mockDB}
	handler := vector.SearchHandler(deps, emb, staticUseSearch(false), staticUseAiTextEmbed(false))
	result, err := handler(context.Background(), buildRequest(t, map[string]any{"query": "find movies"}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.IsError {
		t.Error("expected error result")
	}

	errText := extractText(result)
	if strings.Contains(errText, sensitiveToken) {
		t.Errorf("SECURITY VIOLATION: sensitive token leaked in client error: %q", errText)
	}
}

// extractText extracts the text from the first content item of a tool result.
func extractText(result *mcp.CallToolResult) string {
	if len(result.Content) == 0 {
		return ""
	}
	if tc, ok := result.Content[0].(mcp.TextContent); ok {
		return tc.Text
	}
	return ""
}
