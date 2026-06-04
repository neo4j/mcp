// Copyright (c) "Neo4j"
// Neo4j Sweden AB [http://neo4j.com]

package server_test

import (
	"sort"
	"testing"

	analytics "github.com/neo4j/mcp/internal/analytics/mocks"
	"github.com/neo4j/mcp/internal/config"
	db "github.com/neo4j/mcp/internal/database/mocks"
	"github.com/neo4j/mcp/internal/server"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestToolRegister(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	aService := analytics.NewMockService(ctrl)
	aService.EXPECT().IsEnabled().AnyTimes().Return(true)
	aService.EXPECT().EmitEvent(gomock.Any()).AnyTimes()
	aService.EXPECT().NewStartupEvent(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	// Verify no db calls during tool registration
	mockDB := db.NewMockService(ctrl)
	mockDB.EXPECT().ExecuteReadQuery(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	mockDB.EXPECT().ExecuteWriteQuery(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	mockDB.EXPECT().GetQueryType(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	mockDB.EXPECT().Neo4jRecordsToJSON(gomock.Any()).Times(0)
	t.Run("verifies expected tools are registered", func(t *testing.T) {

		cfg := &config.Config{
			URI:           "bolt://test-host:7687",
			Username:      "neo4j",
			Password:      "password",
			Database:      "neo4j",
			Tools:         config.AvailableTools,
			TransportMode: config.TransportModeStdio,
		}
		s := server.NewNeo4jMCPServer("test-version", cfg, mockDB, aService)

		// Expected tools that should be registered
		// update this number when a tool is added or removed.
		// Current tools: get-schema, read-cypher, write-cypher, list-gds-procedures
		expectedTotalToolsCount := 4

		// Start server and register tools
		err := s.Start()
		if err != nil {
			t.Fatalf("Start() failed: %v", err)
		}
		registeredTools := len(s.MCPServer.ListTools())

		if expectedTotalToolsCount != registeredTools {
			t.Errorf("Expected %d tools, but test configuration shows %d", expectedTotalToolsCount, registeredTools)
		}
	})

	t.Run("should register only readOnly tools when readOnly", func(t *testing.T) {
		cfg := &config.Config{
			URI:           "bolt://test-host:7687",
			Username:      "neo4j",
			Password:      "password",
			Database:      "neo4j",
			ReadOnly:      true,
			Tools:         config.AvailableTools,
			TransportMode: config.TransportModeStdio,
		}
		s := server.NewNeo4jMCPServer("test-version", cfg, mockDB, aService)

		// Expected tools that should be registered
		// update this number when a tool is added or removed.
		// ReadOnly tools: get-schema, read-cypher, list-gds-procedures
		expectedTotalToolsCount := 3

		// Start server and register tools
		err := s.Start()
		if err != nil {
			t.Fatalf("Start() failed: %v", err)
		}
		registeredTools := len(s.MCPServer.ListTools())

		if expectedTotalToolsCount != registeredTools {
			t.Errorf("Expected %d tools, but test configuration shows %d", expectedTotalToolsCount, registeredTools)
		}
	})
	t.Run("should register also write tools when readOnly is set to false", func(t *testing.T) {
		cfg := &config.Config{
			URI:           "bolt://test-host:7687",
			Username:      "neo4j",
			Password:      "password",
			Database:      "neo4j",
			ReadOnly:      false,
			Tools:         config.AvailableTools,
			TransportMode: config.TransportModeStdio,
		}
		s := server.NewNeo4jMCPServer("test-version", cfg, mockDB, aService)

		// Expected tools that should be registered
		// update this number when a tool is added or removed.
		// All tools: get-schema, read-cypher, write-cypher, list-gds-procedures
		expectedTotalToolsCount := 4

		// Start server and register tools
		err := s.Start()
		if err != nil {
			t.Fatalf("Start() failed: %v", err)
		}
		registeredTools := len(s.MCPServer.ListTools())

		if expectedTotalToolsCount != registeredTools {
			t.Errorf("Expected %d tools, but test configuration shows %d", expectedTotalToolsCount, registeredTools)
		}
	})
	t.Run("should only register tools that are specified in config", func(t *testing.T) {
		cfg := &config.Config{
			URI:           "bolt://test-host:7687",
			Username:      "neo4j",
			Password:      "password",
			Database:      "neo4j",
			ReadOnly:      false,
			Tools:         []string{"read-cypher", "get-schema"},
			TransportMode: config.TransportModeStdio,
		}
		s := server.NewNeo4jMCPServer("test-version", cfg, mockDB, aService)

		// Expected tools that should be registered
		expectedTools := []string{"get-schema", "read-cypher"}

		// Start server and register tools
		err := s.Start()
		if err != nil {
			require.NoError(t, err)
		}

		var toolNames []string
		tools := s.MCPServer.ListTools()
		for _, tool := range tools {
			toolNames = append(toolNames, tool.Tool.Name)
		}
		sort.Strings(toolNames)

		assert.Equal(t, expectedTools, toolNames)
	})
	t.Run("should not register write tools when readOnly is enabled even if specified in tools config", func(t *testing.T) {
		cfg := &config.Config{
			URI:           "bolt://test-host:7687",
			Username:      "neo4j",
			Password:      "password",
			Database:      "neo4j",
			ReadOnly:      true,
			Tools:         []string{"write-cypher"},
			TransportMode: config.TransportModeStdio,
		}
		s := server.NewNeo4jMCPServer("test-version", cfg, mockDB, aService)

		// Start server and register tools
		err := s.Start()
		if err != nil {
			require.NoError(t, err)
		}

		assert.Empty(t, s.MCPServer.ListTools())
	})
}
