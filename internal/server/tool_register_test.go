package server_test

import (
	"fmt"
	"testing"

	analytics "github.com/neo4j/mcp/internal/analytics/mocks"
	"github.com/neo4j/mcp/internal/config"
	db "github.com/neo4j/mcp/internal/database/mocks"
	"github.com/neo4j/mcp/internal/server"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"go.uber.org/mock/gomock"
)

func TestToolRegister(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()


	aService := analytics.NewMockService(ctrl)
	aService.EXPECT().EmitEvent(gomock.Any()).AnyTimes()
	aService.EXPECT().NewStartupEvent().AnyTimes()

	t.Run("verifies expected tools are registered", func(t *testing.T) {
		mockDB := getMockedDBService(ctrl, true)
		cfg := &config.Config{
			URI:      "bolt://test-host:7687",
			Username: "neo4j",
			Password: "password",
			Database: "neo4j",
		}
		s := server.NewNeo4jMCPServer("test-version", cfg, mockDB, aService)

		// Expected tools that should be registered
		// update this number when a tool is added or removed.
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

	t.Run("should register only readonly tools when readonly", func(t *testing.T) {
		mockDB := getMockedDBService(ctrl, true)
		cfg := &config.Config{
			URI:      "bolt://test-host:7687",
			Username: "neo4j",
			Password: "password",
			Database: "neo4j",
			ReadOnly: true,
		}
		s := server.NewNeo4jMCPServer("test-version", cfg, mockDB, aService)

		// Expected tools that should be registered
		// update this number when a tool is added or removed.
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
	t.Run("should register also not write tools when readonly is set to false", func(t *testing.T) {
		mockDB := getMockedDBService(ctrl, true)
		cfg := &config.Config{
			URI:      "bolt://test-host:7687",
			Username: "neo4j",
			Password: "password",
			Database: "neo4j",
			ReadOnly: false,
		}
		s := server.NewNeo4jMCPServer("test-version", cfg, mockDB, aService)

		// Expected tools that should be registered
		// update this number when a tool is added or removed.
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

	t.Run("should remove GDS tools if GDS is not present", func(t *testing.T) {
		mockDB := getMockedDBService(ctrl, false)
		cfg := &config.Config{
			URI:      "bolt://test-host:7687",
			Username: "neo4j",
			Password: "password",
			Database: "neo4j",
			ReadOnly: false,
		}
		s := server.NewNeo4jMCPServer("test-version", cfg, mockDB, aService)

		// Expected tools that should be registered
		// update this number when a tool is added or removed.
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
}

// utility to mock the invocation required by VerifyRequirements
func getMockedDBService(ctrl *gomock.Controller, withGDS bool) *db.MockService {
	mockDB := db.NewMockService(ctrl)
	mockDB.EXPECT().VerifyConnectivity(gomock.Any()).Times(1)
	mockDB.EXPECT().ExecuteReadQuery(gomock.Any(), "RETURN 1 as first", gomock.Any()).Times(1).Return([]*neo4j.Record{
		{
			Keys: []string{"first"},
			Values: []any{
				int64(1),
			},
		},
	}, nil)
	checkApocMetaSchemaQuery := "SHOW PROCEDURES YIELD name WHERE name = 'apoc.meta.schema' RETURN count(name) > 0 AS apocMetaSchemaAvailable"
	mockDB.EXPECT().ExecuteReadQuery(gomock.Any(), checkApocMetaSchemaQuery, gomock.Any()).Times(1).Return([]*neo4j.Record{
		{
			Keys: []string{"apocMetaSchemaAvailable"},
			Values: []any{
				bool(true),
			},
		},
	}, nil)
	gdsVersionQuery := "RETURN gds.version() as gdsVersion"
	if withGDS {
		mockDB.EXPECT().ExecuteReadQuery(gomock.Any(), gdsVersionQuery, gomock.Any()).Times(1).Return([]*neo4j.Record{
			{
				Keys: []string{"gdsVersion"},
				Values: []any{
					string("2.22.0"),
				},
			},
		}, nil)
		return mockDB
	}
	mockDB.EXPECT().ExecuteReadQuery(gomock.Any(), gdsVersionQuery, gomock.Any()).Times(1).Return(nil, fmt.Errorf("Unknown function 'gds.version'"))

	return mockDB
}
