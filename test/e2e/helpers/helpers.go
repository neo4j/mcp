//
// Copyright (c) "Neo4j"
// Neo4j Sweden AB [http://neo4j.com]

//go:build e2e

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
	"github.com/neo4j/mcp/internal/config"
	"github.com/neo4j/mcp/internal/database"
	"github.com/neo4j/neo4j-go-driver/v6/neo4j"
	"github.com/stretchr/testify/assert"
)

type UniqueLabel string

// String returns the string representation of the UniqueLabel.
// This implements the fmt.Stringer interface, making it work seamlessly with fmt functions.
func (ul UniqueLabel) String() string {
	return string(ul)
}

// E2ETestContext holds E2E test dependencies focused on database cleanup
// Inspired by TestContext in use by integration tests but adapted for e2e needs.
type E2ETestContext struct {
	ctx           context.Context
	t             *testing.T
	TestID        string
	Service       database.Service
	createdLabels map[string]bool
}

// NewE2ETestContext creates a new E2E test context with automatic cleanup
func NewE2ETestContext(t *testing.T, driver *neo4j.DriverWithContext) *E2ETestContext {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	testID := makeTestID()

	tc := &E2ETestContext{
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
		t.Fatalf("failed to create Neo4j service for E2E test: %v", err)
	}

	tc.Service = databaseService
	return tc
}

// Cleanup removes all test data by deleting nodes with labels created during the test
// This is the primary function for E2E test cleanup
func (tc *E2ETestContext) Cleanup() {
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
			log.Printf("Warning: E2E cleanup failed for label=%s: %v", label, err)
		}
	}
}

// SeedNode creates a test node with a unique label and returns it
func (tc *E2ETestContext) SeedNode(label string, props map[string]any) (UniqueLabel, error) {
	tc.t.Helper()

	if tc.TestID == "" {
		panic("SeedNode: TestID is not set in E2ETestContext. Did you forget to use NewE2ETestContext?")
	}

	uniqueLabel := UniqueLabel(fmt.Sprintf("%s_%s", label, tc.TestID))

	// Track this label for cleanup
	tc.createdLabels[string(uniqueLabel)] = true

	query := fmt.Sprintf("CREATE (n:%s $props) RETURN n", uniqueLabel)
	_, err := tc.Service.ExecuteWriteQuery(tc.ctx, query, map[string]any{"props": props})
	return uniqueLabel, err
}

// GetUniqueLabel returns a unique label for the given base label
func (tc *E2ETestContext) GetUniqueLabel(label string) UniqueLabel {
	if tc.TestID == "" {
		panic("GetUniqueLabel: TestID is not set in E2ETestContext. Did you forget to use NewE2ETestContext?")
	}

	uniqueLabel := UniqueLabel(fmt.Sprintf("%s_%s", label, tc.TestID))

	// Track this label for cleanup
	tc.createdLabels[string(uniqueLabel)] = true

	return uniqueLabel
}

// makeTestID returns a unique test id suitable for tagging resources created by E2E tests
func makeTestID() string {
	id := fmt.Sprintf("e2e-%s", uuid.NewString())
	return strings.ReplaceAll(id, "-", "_")
}

// AssertListContainsJSON checks if a JSON list (at a specific key) contains the expected object.
// responseBody: The raw JSON string returned by your API.
// expectedItem: The struct or map you expect to find.
func (tc *E2ETestContext) AssertJSONListContainsObject(responseBody string, expectedItem interface{}) {
	tc.t.Helper()

	var actualContainer interface{}
	err := json.Unmarshal([]byte(responseBody), &actualContainer)
	assert.NoError(tc.t, err, "Failed to parse response JSON")

	var actualList []interface{}

	var ok bool
	actualList, ok = actualContainer.([]interface{})
	assert.True(tc.t, ok, "Response root is not a list")

	// Marshall and Unmarshal to obtain the same floats/ints conversion.
	expectedBytes, err := json.Marshal(expectedItem)
	assert.NoError(tc.t, err)

	var expectedNormalized interface{}
	err = json.Unmarshal(expectedBytes, &expectedNormalized)
	assert.NoError(tc.t, err)

	assert.Contains(tc.t, actualList, expectedNormalized, "List at '%s' did not contain expected object")
}

func BuildInitializeRequest() mcp.InitializeRequest {
	InitializeRequest := mcp.InitializeRequest{}
	InitializeRequest.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	InitializeRequest.Params.ClientInfo = mcp.Implementation{
		Name:    "test-client",
		Version: "1.0.0",
	}
	return InitializeRequest
}
