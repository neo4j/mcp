//go:build integration

package helpers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/neo4j/mcp/internal/config"
	"github.com/neo4j/mcp/internal/database"
	"github.com/neo4j/mcp/internal/tools"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

type UniqueLabel string

// String returns the string representation of the UniqueLabel.
// This implements the fmt.Stringer interface, making it work seamlessly with fmt functions.
func (ul UniqueLabel) String() string {
	return string(ul)
}

// TestContext holds common test dependencies
type TestContext struct {
	Ctx           context.Context
	T             *testing.T
	TestID        string
	Service       database.Service
	Deps          *tools.ToolDependencies
	createdLabels map[string]bool
	labelMutex    sync.Mutex
}

var (
	cfg       *config.Config
	container testcontainers.Container
	driver    neo4j.DriverWithContext
	once      sync.Once
)

// Start initializes shared resources for integration tests
func Start(ctx context.Context) {
	once.Do(func() {
		startOnce(ctx)
	})
}

func startOnce(ctx context.Context) {
	ctr, boltURI, err := createNeo4jContainer(ctx)
	if err != nil {
		log.Fatalf("failed to start shared neo4j container: %v", err)
	}
	container = ctr

	cfg = &config.Config{
		URI:      boltURI,
		Username: config.GetEnvWithDefault("NEO4J_USERNAME", "neo4j"),
		Password: config.GetEnvWithDefault("NEO4J_PASSWORD", "password"),
		Database: config.GetEnvWithDefault("NEO4J_DATABASE", "neo4j"),
	}

	drv, err := neo4j.NewDriverWithContext(cfg.URI, neo4j.BasicAuth(cfg.Username, cfg.Password, ""))
	if err != nil {
		_ = ctr.Terminate(ctx)
		log.Fatalf("failed to create driver: %v", err)
	}
	driver = drv

	if err := waitForConnectivity(ctx, ctr, &driver); err != nil {
		_ = driver.Close(ctx)
		_ = ctr.Terminate(ctx)
		log.Fatalf("failed to verify connectivity: %v", err)
	}
}

// Close cleans up shared resources used in integration tests
func Close(ctx context.Context) {
	if err := driver.Close(ctx); err != nil {
		log.Printf("Warning: failed to close driver: %v", err)
	}
	if err := container.Terminate(ctx); err != nil {
		log.Printf("Warning: failed to terminate container: %v", err)
	}
}

// NewTestContext creates a new test context with automatic cleanup
func NewTestContext(t *testing.T) *TestContext {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	testID := makeTestID()

	tc := &TestContext{
		Ctx:           ctx,
		T:             t,
		TestID:        testID,
		createdLabels: make(map[string]bool),
	}

	t.Cleanup(func() {
		tc.Cleanup() // Clean up test data
		cancel()     // Release context resources immediately
	})

	svc, err := database.NewNeo4jService(driver)
	if err != nil {
		t.Fatalf("failed to create Neo4j service: %v", err)
	}
	deps := &tools.ToolDependencies{Config: cfg, DBService: svc}

	tc.Service = svc
	tc.Deps = deps

	return tc
}

// Cleanup removes all test data by deleting nodes with labels created during the test
func (tc *TestContext) Cleanup() {
	if tc.Service == nil {
		return // Service wasn't initialized, nothing to clean up
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tc.labelMutex.Lock()
	labels := make([]string, 0, len(tc.createdLabels))
	for label := range tc.createdLabels {
		labels = append(labels, label)
	}
	tc.labelMutex.Unlock()

	// Delete nodes for each unique label
	for _, label := range labels {
		query := fmt.Sprintf("MATCH (n:%s) DETACH DELETE n", label)
		if _, err := tc.Service.ExecuteWriteQuery(
			ctx,
			query,
			map[string]any{},
			cfg.Database,
		); err != nil {
			log.Printf("Warning: cleanup failed for label=%s: %v", label, err)
		}
	}
}

// SeedNode creates a test node with a unique label and returns it.
func (tc *TestContext) SeedNode(label string, props map[string]any) (UniqueLabel, error) {
	tc.T.Helper()

	if tc.TestID == "" {
		panic("SeedNode: TestID is not set in TestContext. Did you forget to use NewTestContext?")
	}

	uniqueLabel := UniqueLabel(fmt.Sprintf("%s_%s", label, tc.TestID))

	// Track this label for cleanup
	tc.labelMutex.Lock()
	tc.createdLabels[string(uniqueLabel)] = true
	tc.labelMutex.Unlock()

	session := (driver).NewSession(tc.Ctx, neo4j.SessionConfig{})
	defer session.Close(tc.Ctx)

	query := fmt.Sprintf("CREATE (n:%s $props) RETURN n", uniqueLabel)
	_, err := session.ExecuteWrite(tc.Ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		result, err := tx.Run(tc.Ctx, query, map[string]any{"props": props})
		if err != nil {
			return nil, err
		}
		_, err = result.Collect(tc.Ctx)
		return nil, err
	})

	return uniqueLabel, err
}

// GetUniqueLabel returns a unique label for the given base label and identifier.
func (tc *TestContext) GetUniqueLabel(label string) UniqueLabel {
	if tc.TestID == "" {
		panic("GetUniqueLabel: TestID is not set in TestContext. Did you forget to use NewTestContext?")
	}

	uniqueLabel := UniqueLabel(fmt.Sprintf("%s_%s", label, tc.TestID))

	// Track this label for cleanup
	tc.labelMutex.Lock()
	tc.createdLabels[string(uniqueLabel)] = true
	tc.labelMutex.Unlock()

	return uniqueLabel
}

// CallTool invokes an MCP tool and returns the response
func (tc *TestContext) CallTool(handler func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error), args map[string]any) *mcp.CallToolResult {
	tc.T.Helper()

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: args,
		},
	}

	res, err := handler(tc.Ctx, req)
	if err != nil {
		tc.T.Fatalf("tool call failed: %v", err)
	}
	if res == nil {
		tc.T.Fatal("tool returned nil response")
	}
	if res.IsError {
		tc.T.Fatalf("tool returned error: %+v", res)
	}

	return res
}

// ParseJSONResponse parses JSON response into the provided interface
func (tc *TestContext) ParseJSONResponse(res *mcp.CallToolResult, v any) {
	tc.T.Helper()

	if len(res.Content) == 0 {
		tc.T.Fatal("response has no content")
	}

	textContent, ok := mcp.AsTextContent(res.Content[0])
	if !ok {
		tc.T.Fatalf("expected TextContent, got %T", res.Content[0])
	}

	if err := json.Unmarshal([]byte(textContent.Text), v); err != nil {
		tc.T.Fatalf("failed to parse JSON response: %v\nraw: %s", err, textContent.Text)
	}
}

// VerifyNodeInDB verifies that a node exists in the database with the given properties.
// The label parameter should be the unique label (e.g., "Person_test_abc123").
func (tc *TestContext) VerifyNodeInDB(label UniqueLabel, props map[string]any) *neo4j.Record {
	tc.T.Helper()

	session := (driver).NewSession(tc.Ctx, neo4j.SessionConfig{})
	defer session.Close(tc.Ctx)

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
	result, err := session.ExecuteRead(tc.Ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		res, err := tx.Run(tc.Ctx, query, props)
		if err != nil {
			return nil, err
		}
		return res.Collect(tc.Ctx)
	})
	if err != nil {
		tc.T.Fatalf("failed to verify node in DB: %v", err)
	}

	records, ok := result.([]*neo4j.Record)
	if !ok || len(records) != 1 {
		tc.T.Fatalf("expected 1 record in DB, got %d", len(records))
	}

	return records[0]
}

// AssertNodeProperties validates node properties match expected values
func AssertNodeProperties(t *testing.T, node map[string]any, expectedProps map[string]any) {
	t.Helper()

	props, ok := node["Props"].(map[string]any)
	if !ok {
		t.Fatalf("expected 'Props' to be a map, got %T: %+v", node["Props"], node)
	}

	for key, expectedVal := range expectedProps {
		actualVal, exists := props[key]
		if !exists {
			t.Errorf("property %q not found in node", key)
			continue
		}

		if actualVal != expectedVal {
			t.Errorf("property %q: expected %v (type=%T), got %v (type=%T)",
				key, expectedVal, expectedVal, actualVal, actualVal)
		}
	}
}

// AssertNodeHasLabel checks if a node has a specific label
func AssertNodeHasLabel(t *testing.T, node map[string]any, expectedLabel UniqueLabel) {
	t.Helper()

	labels, ok := node["Labels"].([]any)
	if !ok {
		t.Fatalf("expected 'Labels' to be a slice, got %T", node["Labels"])
	}

	for _, label := range labels {
		if labelStr, ok := label.(string); ok && labelStr == string(expectedLabel) {
			return
		}
	}

	t.Errorf("expected node to have label %q, got labels=%v", expectedLabel, labels)
}

// AssertSchemaHasNodeType checks if the schema contains a node type with expected properties
func AssertSchemaHasNodeType(t *testing.T, schemaMap map[string]map[string]any, label UniqueLabel, expectedProps []string) {
	t.Helper()

	schema, ok := schemaMap[string(label)]
	if !ok {
		t.Errorf("expected schema to contain '%s' label", label)
		return
	}

	if schema["type"] != "node" {
		t.Errorf("expected %s type to be 'node', got %v", label, schema["type"])
	}

	props, ok := schema["properties"].(map[string]any)
	if !ok {
		t.Errorf("expected %s to have properties", label)
		return
	}

	for _, prop := range expectedProps {
		if _, exists := props[prop]; !exists {
			t.Errorf("expected %s to have '%s' property", label, prop)
		}
	}
}

// makeTestID returns a unique test id suitable for tagging resources created by tests.
func makeTestID() string {
	id := fmt.Sprintf("test-%s", uuid.NewString())
	return strings.ReplaceAll(id, "-", "_")
}

// waitForConnectivity waits for Neo4j connectivity with exponential backoff.
func waitForConnectivity(ctx context.Context, ctr testcontainers.Container, driver *neo4j.DriverWithContext) error {
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	backoff := 100 * time.Millisecond
	maxBackoff := 2 * time.Second

	var lastErr error
	for {
		if err := (*driver).VerifyConnectivity(ctx); err == nil {
			return nil
		} else {
			lastErr = err
		}

		if ctx.Err() != nil {
			break
		}

		time.Sleep(backoff)
		backoff *= 2
		if backoff > maxBackoff {
			backoff = maxBackoff
		}
	}

	var logs string
	if ctr != nil {
		rc, err := ctr.Logs(context.Background())
		if err == nil && rc != nil {
			b, rerr := io.ReadAll(rc)
			_ = rc.Close()
			if rerr == nil {
				logs = string(b)
			}
		}
	}

	if logs != "" {
		return fmt.Errorf("neo4j connectivity not ready: %v\ncontainer logs:\n%s", lastErr, logs)
	}
	return fmt.Errorf("neo4j connectivity not ready: %v", lastErr)
}

// createNeo4jContainer starts a Neo4j container for testing
func createNeo4jContainer(ctx context.Context) (testcontainers.Container, string, error) {
	req := testcontainers.ContainerRequest{
		Image:        config.GetEnvWithDefault("NEO4J_IMAGE", "neo4j:5.24.2-community"),
		ExposedPorts: []string{"7687/tcp"},
		Env: map[string]string{
			"NEO4J_AUTH":        fmt.Sprintf("%s/%s", config.GetEnvWithDefault("NEO4J_USERNAME", "neo4j"), config.GetEnvWithDefault("NEO4J_PASSWORD", "password")),
			"NEO4JLABS_PLUGINS": config.GetEnvWithDefault("NEO4JLABS_PLUGINS", `["apoc","graph-data-science"]`),
		},
		WaitingFor: wait.ForListeningPort("7687/tcp").WithStartupTimeout(119 * time.Second),
	}

	ctr, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return nil, "", err
	}

	host, err := ctr.Host(ctx)
	if err != nil {
		_ = ctr.Terminate(ctx)
		return nil, "", err
	}
	port, err := ctr.MappedPort(ctx, "7687/tcp")
	if err != nil {
		_ = ctr.Terminate(ctx)
		return nil, "", err
	}

	boltURI := fmt.Sprintf("bolt://%s:%s", host, port.Port())

	return ctr, boltURI, nil
}
