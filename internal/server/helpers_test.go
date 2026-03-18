// Copyright (c) "Neo4j"
// Neo4j Sweden AB [http://neo4j.com]

package server

import (
	"net/http"
	"testing"

	"github.com/mark3labs/mcp-go/server"
	analytics_mocks "github.com/neo4j/mcp/internal/analytics/mocks"
	"github.com/neo4j/mcp/internal/config"
	db_mocks "github.com/neo4j/mcp/internal/database/mocks"
	"go.uber.org/mock/gomock"
)

// mockNeo4jMCPServer creates a Neo4jMCPServer with mock dependencies for use in tests.
func mockNeo4jMCPServer(t *testing.T) *Neo4jMCPServer {
	t.Helper()
	ctrl := gomock.NewController(t)

	cfg := &config.Config{
		URI:           "bolt://localhost:7687",
		Username:      "neo4j",
		Password:      "password",
		Database:      "neo4j",
		TransportMode: config.TransportModeHTTP,
		Telemetry:     false,
	}

	mockDBService := db_mocks.NewMockService(ctrl)
	mockAnalyticsService := analytics_mocks.NewMockService(ctrl)

	mcpServer := server.NewMCPServer("test-server", "1.0.0")

	return &Neo4jMCPServer{
		MCPServer:    mcpServer,
		config:       cfg,
		dbService:    mockDBService,
		anService:    mockAnalyticsService,
		version:      "1.0.0",
		gdsInstalled: false,
	}
}

// mockHandler is a simple HTTP handler that always returns 200 OK.
func mockHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})
}
