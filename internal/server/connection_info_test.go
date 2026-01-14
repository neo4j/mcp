package server

import (
	"context"
	"testing"

	analytics "github.com/neo4j/mcp/internal/analytics/mocks"
	"github.com/neo4j/mcp/internal/config"
	db "github.com/neo4j/mcp/internal/database/mocks"
	"go.uber.org/mock/gomock"
)

func TestCollectConnectionInfo_MissingAuth(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	cfg := &config.Config{
		TransportMode: config.TransportModeHTTP,
		Database:      "neo4j",
	}

	mockDB := db.NewMockService(ctrl)
	// DB should NOT be called if auth is missing - no expectations set
	// If DB is called, the test will fail due to unexpected call

	mockAnalytics := analytics.NewMockService(ctrl)

	srv := &Neo4jMCPServer{
		config:    cfg,
		dbService: mockDB,
		anService: mockAnalytics,
	}

	// Context without auth credentials
	ctx := context.Background()

	// Call collectConnectionInfo
	connInfo := srv.collectConnectionInfo(ctx)

	// Should return "unknown" values for all fields
	if connInfo.Neo4jVersion != "unknown" {
		t.Errorf("Expected Neo4jVersion to be 'unknown', got '%s'", connInfo.Neo4jVersion)
	}
	if connInfo.Edition != "unknown" {
		t.Errorf("Expected Edition to be 'unknown', got '%s'", connInfo.Edition)
	}
	if len(connInfo.CypherVersion) != 1 || connInfo.CypherVersion[0] != "unknown" {
		t.Errorf("Expected CypherVersion to be ['unknown'], got %v", connInfo.CypherVersion)
	}

	// The test passes if mockDB.ExecuteReadQuery was NOT called (gomock will verify this)
	mockDB.EXPECT().ExecuteReadQuery(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
}

func TestCollectConnectionInfo_STDIOModeIgnoresAuth(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	cfg := &config.Config{
		TransportMode: config.TransportModeStdio,
		Database:      "neo4j",
	}

	mockDB := db.NewMockService(ctrl)
	// In STDIO mode, DB query should proceed regardless of auth in context
	mockDB.EXPECT().
		ExecuteReadQuery(gomock.Any(), "CALL dbms.components()", gomock.Any()).
		Times(1).
		Return(nil, nil) // Return empty result for this test

	mockAnalytics := analytics.NewMockService(ctrl)

	srv := &Neo4jMCPServer{
		config:    cfg,
		dbService: mockDB,
		anService: mockAnalytics,
	}

	// Context without auth credentials (doesn't matter in STDIO mode)
	ctx := context.Background()

	// Call collectConnectionInfo - should proceed with DB query
	connInfo := srv.collectConnectionInfo(ctx)

	// Verify default "unknown" value is returned (since we returned empty records)
	if connInfo.Neo4jVersion != "unknown" {
		t.Errorf("Expected Neo4jVersion to be 'unknown' for empty records, got '%s'", connInfo.Neo4jVersion)
	}
}
