package helpers

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/neo4j/mcp/internal/database"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
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

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
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

	databaseService, err := database.NewNeo4jService(*driver, "neo4j")
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

// VerifyNodeInDB verifies that a node exists in the database with the given properties
func (tc *E2ETestContext) VerifyNodeInDB(label UniqueLabel, props map[string]any) *neo4j.Record {
	tc.t.Helper()

	// Build WHERE clause dynamically, excluding test_id from verification
	whereClauses := []string{}
	queryParams := make(map[string]any)

	for key, value := range props {
		if key != "test_id" { // Skip test_id in verification
			whereClauses = append(whereClauses, fmt.Sprintf("n.%s = $%s", key, key))
			queryParams[key] = value
		}
	}

	whereClause := ""
	if len(whereClauses) > 0 {
		whereClause = " WHERE " + strings.Join(whereClauses, " AND ")
	}

	query := fmt.Sprintf("MATCH (n:%s)%s RETURN n", label, whereClause)
	records, err := tc.Service.ExecuteReadQuery(tc.ctx, query, queryParams)
	if err != nil {
		tc.t.Fatalf("E2E: failed to verify node in DB: %v", err)
	}
	if len(records) != 1 {
		tc.t.Fatalf("E2E: expected 1 record in DB, got %d", len(records))
	}

	return records[0]
}

// AssertNodeProperties validates node properties match expected values
func (tc *E2ETestContext) AssertNodeProperties(node map[string]any, expectedProps map[string]any) {
	tc.t.Helper()

	props, ok := node["Props"].(map[string]any)
	if !ok {
		tc.t.Fatalf("E2E: expected 'Props' to be a map, got %T: %+v", node["Props"], node)
	}

	for key, expectedVal := range expectedProps {
		if key == "test_id" {
			continue // Skip test_id in property assertions
		}

		actualVal, exists := props[key]
		if !exists {
			tc.t.Errorf("E2E: property %q not found in node", key)
			continue
		}

		if actualVal != expectedVal {
			tc.t.Errorf("E2E: property %q: expected %v (type=%T), got %v (type=%T)",
				key, expectedVal, expectedVal, actualVal, actualVal)
		}
	}
}

// AssertNodeHasLabel checks if a node has a specific label
func (tc *E2ETestContext) AssertNodeHasLabel(node map[string]any, expectedLabel UniqueLabel) {
	tc.t.Helper()

	labels, ok := node["Labels"].([]any)
	if !ok {
		tc.t.Fatalf("E2E: expected 'Labels' to be a slice, got %T", node["Labels"])
	}

	for _, label := range labels {
		if labelStr, ok := label.(string); ok && labelStr == string(expectedLabel) {
			return
		}
	}

	tc.t.Errorf("E2E: expected node to have label %q, got labels=%v", expectedLabel, labels)
}

// makeTestID returns a unique test id suitable for tagging resources created by E2E tests
func makeTestID() string {
	id := fmt.Sprintf("e2e-%s", uuid.NewString())
	return strings.ReplaceAll(id, "-", "_")
}

func (tc *E2ETestContext) BuildServer(t *testing.T) (string, func(), error) {
	tc.t.Helper()
	// Create temporary directory for the build
	tmpDir := os.TempDir()

	// Create a unique subdirectory for this test run
	buildDir, err := os.MkdirTemp(tmpDir, "mcp-server-test-*")
	if err != nil {
		return "", nil, err
	}

	// Define cleanup function
	cleanup := func() {
		if err := os.RemoveAll(buildDir); err != nil {
			t.Logf("failed to cleanup build directory: %v", err)
		}
	}

	// Define binary path
	binaryName := "neo4j-mcp"
	binaryPath := filepath.Join(buildDir, binaryName)

	// Get the project root directory (go up from test/e2e/)
	projectRoot := filepath.Join("..", "..")

	// Build the server binary
	cmd := exec.Command("go", "build", "-o", binaryPath, ".")
	cmd.Dir = filepath.Join(projectRoot, "cmd", "neo4j-mcp")
	cmd.Env = os.Environ() // Use current environment

	// Capture build output for debugging
	output, err := cmd.CombinedOutput()
	if err != nil {
		cleanup()
		return "", nil, err
	}

	t.Logf("Built server binary at: %s", binaryPath)
	if len(output) > 0 {
		t.Logf("Build output: %s", string(output))
	}

	// Verify the binary was created, if not cleanup and return
	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		cleanup()
		return "", nil, err
	}

	return binaryPath, cleanup, nil
}

func (tc *E2ETestContext) BuildInitializeRequest() mcp.InitializeRequest {
	tc.t.Helper()
	InitializeRequest := mcp.InitializeRequest{}
	InitializeRequest.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	InitializeRequest.Params.ClientInfo = mcp.Implementation{
		Name:    "test-client",
		Version: "1.0.0",
	}
	return InitializeRequest
}
