//go:build integration

package integration

import (
	"strings"
	"testing"

	"github.com/neo4j/mcp/internal/database"
	"github.com/neo4j/mcp/internal/tools/gds"
	"github.com/neo4j/mcp/test/integration/container_runner"
	"github.com/neo4j/mcp/test/integration/helpers"
)

func TestListGdsProcedures(t *testing.T) {
	t.Parallel()
	driver := container_runner.GetContainerDriver()
	databaseService, err := database.NewNeo4jService(*driver, "neo4j")
	if err != nil {
		t.Fatalf("failed to create Neo4j service: %v", err)
	}
	tc := helpers.NewTestContext(t, databaseService)

	listGds := gds.ListGdsProceduresHandler(tc.Deps)
	res := tc.CallTool(listGds, nil)

	var procedures []map[string]any
	tc.ParseJSONResponse(res, &procedures)

	// Should have GDS procedures since we enabled the plugin
	if len(procedures) == 0 {
		t.Fatal("Expected GDS procedures to be available, but got empty list")
	}

	// Verify the structure of returned procedures
	firstProc := procedures[0]

	// Check that expected fields exist
	if _, ok := firstProc["name"]; !ok {
		t.Error("Expected 'name' field in procedure")
	}
	if _, ok := firstProc["description"]; !ok {
		t.Error("Expected 'description' field in procedure")
	}
	if _, ok := firstProc["signature"]; !ok {
		t.Error("Expected 'signature' field in procedure")
	}
	if procType, ok := firstProc["type"]; !ok {
		t.Error("Expected 'type' field in procedure")
	} else if procType != "procedure" {
		t.Errorf("Expected type='procedure', got %v", procType)
	}

	// Verify procedures are filtered correctly (streaming procedures, no estimates)
	for _, proc := range procedures {
		name, ok := proc["name"].(string)
		if !ok {
			t.Errorf("Expected name to be string, got %T", proc["name"])
			continue
		}

		if !strings.Contains(name, "stream") {
			t.Errorf("Expected all procedures to contain 'stream', but found: %s", name)
		}
		if strings.Contains(name, "estimate") {
			t.Errorf("Expected no 'estimate' procedures, but found: %s", name)
		}
	}

	t.Logf("Found %d GDS streaming procedures", len(procedures))
}

func TestListGdsProcedures_KnownProcedures(t *testing.T) {
	t.Parallel()
	driver := container_runner.GetContainerDriver()
	databaseService, err := database.NewNeo4jService(*driver, "neo4j")
	if err != nil {
		t.Fatalf("failed to create Neo4j service: %v", err)
	}
	tc := helpers.NewTestContext(t, databaseService)

	listGds := gds.ListGdsProceduresHandler(tc.Deps)
	res := tc.CallTool(listGds, nil)

	var procedures []map[string]any
	tc.ParseJSONResponse(res, &procedures)

	// Build a map of procedure names for easy lookup
	procNames := make(map[string]bool)
	for _, proc := range procedures {
		if name, ok := proc["name"].(string); ok {
			procNames[name] = true
		}
	}

	// Check for some common GDS streaming procedures
	expectedProcedures := []string{
		"gds.graph.list",      // List graph projections (streaming)
		"gds.degree.stream",   // Degree centrality
		"gds.pageRank.stream", // PageRank
	}

	foundCount := 0
	for _, expected := range expectedProcedures {
		if procNames[expected] {
			foundCount++
			t.Logf("✓ Found expected procedure: %s", expected)
		}
	}

	if foundCount == 0 {
		t.Errorf("Expected to find at least some common GDS procedures like %v, but found none", expectedProcedures)
		t.Logf("Available procedures: %v", procNames)
	}
}
