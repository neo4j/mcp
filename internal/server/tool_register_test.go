package server_test

import (
	"testing"

	"github.com/neo4j/mcp/internal/config"
	"github.com/neo4j/mcp/internal/database/mocks"
	"github.com/neo4j/mcp/internal/server"
	"go.uber.org/mock/gomock"
)

func TestGetAllTools(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	cfg := &config.Config{
		URI:      "bolt://test-host:7687",
		Username: "neo4j",
		Password: "password",
		Database: "neo4j",
	}

	mockDB := mocks.NewMockService(ctrl)

	t.Run("verifies expected tools are registered", func(t *testing.T) {
		s := server.NewNeo4jMCPServer("test-version", cfg, mockDB)

		// Expected tools that should be registered
		expectedTools := []string{
			"get-schema",
			"read-cypher",
			"write-cypher",
			"list-gds-procedures",
		}

		// Register tools
		err := s.RegisterTools()
		if err != nil {
			t.Fatalf("RegisterTools() failed: %v", err)
		}
		registeredTools := len(s.MCPServer.ListTools())

		if len(expectedTools) != registeredTools {
			t.Errorf("Expected 4 tools, but test configuration shows %d", len(expectedTools))
		}
	})

}
