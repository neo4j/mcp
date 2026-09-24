// Copyright (c) "Neo4j"
// Neo4j Sweden AB [http://neo4j.com]

package server_test

import (
	"context"
	"sort"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	analytics "github.com/neo4j/mcp/internal/analytics/mocks"
	"github.com/neo4j/mcp/internal/config"
	db "github.com/neo4j/mcp/internal/database/mocks"
	"github.com/neo4j/mcp/internal/server"
	"github.com/neo4j/neo4j-go-driver/v6/neo4j"
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
	aService.EXPECT().NewConnectionInitializedEvent(gomock.Any()).AnyTimes()
	// Client handshake required for tool registration.
	mockDB := db.NewMockService(ctrl)
	mockDB.EXPECT().ExecuteReadQuery(gomock.Any(), "RETURN 1 as first", gomock.Any()).AnyTimes().Return([]*neo4j.Record{
		{Keys: []string{"first"},
			Values: []any{int64(1)},
		},
	}, nil)
	mockDB.EXPECT().ExecuteReadQuery(gomock.Any(), "SHOW PROCEDURES YIELD name WHERE name = 'apoc.meta.schema' RETURN count(name) > 0 AS apocMetaSchemaAvailable", gomock.Any()).AnyTimes().Return([]*neo4j.Record{
		{Keys: []string{"apocMetaSchemaAvailable"},
			Values: []any{bool(true)},
		},
	}, nil)
	mockDB.EXPECT().ExecuteReadQuery(gomock.Any(), "RETURN gds.version() as gdsVersion", gomock.Any()).AnyTimes().Return([]*neo4j.Record{
		{Keys: []string{"gdsVersion"}, Values: []any{string("2.22.0")}},
	}, nil)
	mockDB.EXPECT().ExecuteReadQuery(gomock.Any(), "CALL dbms.components()", gomock.Any()).AnyTimes()
	mockDB.EXPECT().ExecuteWriteQuery(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	mockDB.EXPECT().GetQueryType(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	mockDB.EXPECT().Neo4jRecordsToJSON(gomock.Any()).Times(0)
	t.Run("verifies expected tools are registered", func(t *testing.T) {
		withFreshStdin(t)

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
		registeredTools := len(listRegisteredTools(t, s))

		if expectedTotalToolsCount != registeredTools {
			t.Errorf("Expected %d tools, but test configuration shows %d", expectedTotalToolsCount, registeredTools)
		}
	})

	t.Run("should register only readOnly tools when readOnly", func(t *testing.T) {
		withFreshStdin(t)
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
		registeredTools := len(listRegisteredTools(t, s))

		if expectedTotalToolsCount != registeredTools {
			t.Errorf("Expected %d tools, but test configuration shows %d", expectedTotalToolsCount, registeredTools)
		}
	})
	t.Run("should register also write tools when readOnly is set to false", func(t *testing.T) {
		withFreshStdin(t)
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
		registeredTools := len(listRegisteredTools(t, s))

		if expectedTotalToolsCount != registeredTools {
			t.Errorf("Expected %d tools, but test configuration shows %d", expectedTotalToolsCount, registeredTools)
		}
	})
	t.Run("should only register tools that are specified in config", func(t *testing.T) {
		withFreshStdin(t)
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
		for _, tool := range listRegisteredTools(t, s) {
			toolNames = append(toolNames, tool.Name)
		}
		sort.Strings(toolNames)

		assert.Equal(t, expectedTools, toolNames)
	})
	t.Run("should not register write tools when readOnly is enabled even if specified in tools config", func(t *testing.T) {
		withFreshStdin(t)
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

		assert.Empty(t, listRegisteredTools(t, s))
	})
}

// listRegisteredTools connects an in-process client to s.MCPServer and returns its advertised tools.
func listRegisteredTools(t *testing.T, s *server.Neo4jMCPServer) []*mcp.Tool {
	t.Helper()

	ctx := context.Background()
	session, err := connectInProcessClient(ctx, t, s.MCPServer)
	if err != nil {
		t.Fatalf("failed to connect in-process client: %v", err)
	}
	defer session.Close()

	listToolsResponse, err := session.ListTools(ctx, &mcp.ListToolsParams{})
	if err != nil {
		t.Fatalf("failed to list tools: %v", err)
	}
	return listToolsResponse.Tools
}
