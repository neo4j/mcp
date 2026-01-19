//go:build integration

package helpers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/mark3labs/mcp-go/mcp"
	analytics "github.com/neo4j/mcp/internal/analytics/mocks"
	"github.com/neo4j/mcp/internal/config"
	"github.com/neo4j/mcp/internal/database"
	"github.com/neo4j/mcp/internal/tools"
	"github.com/neo4j/neo4j-go-driver/v6/neo4j"
	"go.uber.org/mock/gomock"
)

type UniqueLabel string

// String returns the string representation of the UniqueLabel.
// This implements the fmt.Stringer interface, making it work seamlessly with fmt functions.
func (ul UniqueLabel) String() string {
	return string(ul)
}

// TestContext holds common test dependencies
type TestContext struct {
	ctx              context.Context
	t                *testing.T
	TestID           string
	Service          database.Service
	Deps             *tools.ToolDependencies
	createdLabels    map[string]bool
	AnalyticsService *analytics.MockService
}

// NewTestContext creates a new test context with automatic cleanup
func NewTestContext(t *testing.T, driver *neo4j.Driver) *TestContext {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	testID := makeTestID()

	tc := &TestContext{
		ctx:           ctx,
		t:             t,
		TestID:        testID,
		createdLabels: make(map[string]bool),
	}

	t.Cleanup(func() {
		tc.Cleanup() // Clean up test data
		cancel()     // Release context resources immediately
	})

	databaseService, err := database.NewNeo4jService(*driver, "neo4j", config.TransportModeStdio, "test-version")
	if err != nil {
		t.Fatalf("failed to create Neo4j service: %v", err)
	}

	analyticsService := getAnalyticsMock(t)
	deps := &tools.ToolDependencies{
		DBService:        databaseService,
		AnalyticsService: analyticsService,
	}

	tc.AnalyticsService = analyticsService
	tc.Service = databaseService
	tc.Deps = deps

	return tc
}

// getAnalyticsMock is used to mock the analytics service, for integration test purpose.
func getAnalyticsMock(t *testing.T) *analytics.MockService {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	analyticsService := analytics.NewMockService(ctrl)
	analyticsService.EXPECT().IsEnabled().AnyTimes().Return(true)
	analyticsService.EXPECT().EmitEvent(gomock.Any()).AnyTimes()
	analyticsService.EXPECT().Disable().AnyTimes()
	analyticsService.EXPECT().Enable().AnyTimes()
	analyticsService.EXPECT().NewGDSProjCreatedEvent().AnyTimes()
	analyticsService.EXPECT().NewStartupEvent(gomock.Any()).AnyTimes()
	analyticsService.EXPECT().NewToolEvent(gomock.Any(), gomock.Any()).AnyTimes()
	analyticsService.EXPECT().NewConnectionInitializedEvent(gomock.Any()).AnyTimes()

	return analyticsService
}

// Cleanup removes all test data by deleting nodes with labels created during the test
func (tc *TestContext) Cleanup() {
	if tc.Service == nil {
		return // Service wasn't initialized, nothing to clean up
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	labels := make([]string, 0, len(tc.createdLabels))
	for label := range tc.createdLabels {
		labels = append(labels, label)
	}

	// Delete nodes for each unique label
	for _, label := range labels {
		query := fmt.Sprintf("MATCH (n:%s) DETACH DELETE n", label)
		if _, err := tc.Service.ExecuteWriteQuery(
			ctx,
			query,
			map[string]any{},
		); err != nil {
			log.Printf("Warning: cleanup failed for label=%s: %v", label, err)
		}
	}
}

// SeedNode creates a test node with a unique label and returns it.
func (tc *TestContext) SeedNode(label string, props map[string]any) (UniqueLabel, error) {
	tc.t.Helper()

	if tc.TestID == "" {
		panic("SeedNode: TestID is not set in TestContext. Did you forget to use NewTestContext?")
	}

	uniqueLabel := UniqueLabel(fmt.Sprintf("%s_%s", label, tc.TestID))

	// Track this label for cleanup
	tc.createdLabels[string(uniqueLabel)] = true

	query := fmt.Sprintf("CREATE (n:%s $props) RETURN n", uniqueLabel)
	_, err := tc.Service.ExecuteWriteQuery(tc.ctx, query, map[string]any{"props": props})
	return uniqueLabel, err

}

// GetUniqueLabel returns a unique label for the given base label and identifier.
func (tc *TestContext) GetUniqueLabel(label string) UniqueLabel {
	if tc.TestID == "" {
		panic("GetUniqueLabel: TestID is not set in TestContext. Did you forget to use NewTestContext?")
	}

	uniqueLabel := UniqueLabel(fmt.Sprintf("%s_%s", label, tc.TestID))

	// Track this label for cleanup

	tc.createdLabels[string(uniqueLabel)] = true

	return uniqueLabel
}

// CallTool invokes an MCP tool and returns the response
func (tc *TestContext) CallTool(handler func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error), args map[string]any) *mcp.CallToolResult {
	tc.t.Helper()

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: args,
		},
	}

	res, err := handler(tc.ctx, req)
	if err != nil {
		tc.t.Fatalf("tool call failed: %v", err)
		return nil
	}
	if res == nil {
		tc.t.Fatal("tool returned nil response")
		return nil
	}
	if res.IsError {
		tc.t.Fatalf("tool returned error: %+v", res)
		return nil
	}

	return res
}

// Similar to CallTool but returns the error to assert error handlings, if mcp.CallToolResult.isError is false then fails
func (tc *TestContext) GetToolError(handler func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error), args map[string]any) string {
	tc.t.Helper()

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: args,
		},
	}

	res, err := handler(tc.ctx, req)
	if err != nil {
		tc.t.Fatalf("tool call failed: %v", err)
		return ""
	}
	if res == nil {
		tc.t.Fatal("tool returned nil response")
		return ""
	}
	if !res.IsError {
		tc.t.Fatal("no error returned")
		return ""
	}

	textContent, ok := mcp.AsTextContent(res.Content[0])
	if !ok {
		tc.t.Fatalf("expected error as TextContent, got %T", res.Content[0])
		return ""
	}
	return textContent.Text
}

// ParseJSONResponse parses JSON response into the provided interface
func (tc *TestContext) ParseJSONResponse(res *mcp.CallToolResult, v any) {
	tc.t.Helper()

	if len(res.Content) == 0 {
		tc.t.Fatal("response has no content")
	}

	textContent, ok := mcp.AsTextContent(res.Content[0])
	if !ok {
		tc.t.Fatalf("expected TextContent, got %T", res.Content[0])
	}

	if err := json.Unmarshal([]byte(textContent.Text), v); err != nil {
		tc.t.Fatalf("failed to parse JSON response: %v\nraw: %s", err, textContent.Text)
	}
}

// ParseTextResponse parses Text response and returns a string
func (tc *TestContext) ParseTextResponse(res *mcp.CallToolResult) string {
	tc.t.Helper()

	if len(res.Content) == 0 {
		tc.t.Fatal("response has no content")
	}

	textContent, ok := mcp.AsTextContent(res.Content[0])
	if !ok {
		tc.t.Fatalf("expected TextContent, got %T", res.Content[0])
	}
	return textContent.Text
}

// VerifyNodeInDB verifies that a node exists in the database with the given properties.
// The label parameter should be the unique label (e.g., "Person_test_abc123").
func (tc *TestContext) VerifyNodeInDB(label UniqueLabel, props map[string]any) *neo4j.Record {
	tc.t.Helper()

	// Build WHERE clause dynamically
	whereClauses := []string{}
	for key := range props {
		whereClauses = append(whereClauses, fmt.Sprintf("n.%s = $%s", key, key))
	}
	whereClause := ""
	if len(whereClauses) > 0 {
		whereClause = " WHERE " + strings.Join(whereClauses, " AND ")
	}

	query := fmt.Sprintf("MATCH (n:%s)%s RETURN n", label, whereClause)
	records, err := tc.Service.ExecuteReadQuery(tc.ctx, query, props)
	if err != nil {
		tc.t.Fatalf("failed to verify node in DB: %v", err)
	}
	if len(records) != 1 {
		tc.t.Fatalf("expected 1 record in DB, got %d", len(records))
	}

	return records[0]
}

// AssertNodeProperties validates node properties match expected values
func (tc *TestContext) AssertNodeProperties(node map[string]any, expectedProps map[string]any) {
	tc.t.Helper()

	props, ok := node["Props"].(map[string]any)
	if !ok {
		tc.t.Fatalf("expected 'Props' to be a map, got %T: %+v", node["Props"], node)
	}

	for key, expectedVal := range expectedProps {
		actualVal, exists := props[key]
		if !exists {
			tc.t.Errorf("property %q not found in node", key)
			continue
		}

		if actualVal != expectedVal {
			tc.t.Errorf("property %q: expected %v (type=%T), got %v (type=%T)",
				key, expectedVal, expectedVal, actualVal, actualVal)
		}
	}
}

// AssertNodeHasLabel checks if a node has a specific label
func (tc *TestContext) AssertNodeHasLabel(node map[string]any, expectedLabel UniqueLabel) {
	tc.t.Helper()

	labels, ok := node["Labels"].([]any)
	if !ok {
		tc.t.Fatalf("expected 'Labels' to be a slice, got %T", node["Labels"])
	}

	for _, label := range labels {
		if labelStr, ok := label.(string); ok && labelStr == string(expectedLabel) {
			return
		}
	}

	tc.t.Errorf("expected node to have label %q, got labels=%v", expectedLabel, labels)
}

// makeTestID returns a unique test id suitable for tagging resources created by tests.
func makeTestID() string {
	id := fmt.Sprintf("test-%s", uuid.NewString())
	return strings.ReplaceAll(id, "-", "_")
}
