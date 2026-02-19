// Copyright (c) "Neo4j"
// Neo4j Sweden AB [http://neo4j.com]

//go:build integration

package integration

import (
	"strings"
	"testing"

	"github.com/neo4j/mcp/internal/tools/gds"
	"github.com/neo4j/mcp/test/integration/helpers"
)

func TestListGdsProcedures(t *testing.T) {
	t.Parallel()

	tc := helpers.NewTestContext(t, dbs.GetDriver())

	listGds := gds.ListGdsProceduresHandler(tc.Deps)
	res := tc.CallTool(listGds, nil)

	var procedures []map[string]any
	tc.ParseJSONResponse(res, &procedures)
	t.Run("should return some procedures", func(t *testing.T) {
		// Should have GDS procedures since we enabled the plugin
		if len(procedures) == 0 {
			t.Fatal("Expected GDS procedures to be available, but got empty list")
		}

	})
	t.Run("should check the format returned", func(t *testing.T) {
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
	})
	t.Run("should verify that procedures are filtered correctly (streaming procedures, no estimates)", func(t *testing.T) {
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
	})
	t.Run("should check that some of the expected GDS procedure are returned", func(t *testing.T) {
		// Build a map of procedure names for easy lookup
		procNames := make(map[string]bool)
		for _, proc := range procedures {
			if name, ok := proc["name"].(string); ok {
				procNames[name] = true
			}
		}

		// Check for some common GDS streaming procedures
		expectedProcedures := []string{
			"gds.betweenness.stream",
			"gds.degree.stream",
			"gds.pageRank.stream",
			"gds.eigenvector.stream",
			"gds.shortestPath.dijkstra.stream",
		}

		foundCount := 0
		for _, expected := range expectedProcedures {
			if procNames[expected] {
				foundCount++
				continue
			}
			t.Fatalf("Expected to find some common GDS procedures: %v, but %s was not found", expectedProcedures, expected)
		}
	})
}
