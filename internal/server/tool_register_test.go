package server_test

import (
	"io"
	"testing"

	"github.com/neo4j/mcp/internal/config"
	"github.com/neo4j/mcp/internal/database/mocks"
	"github.com/neo4j/mcp/internal/logger"
	"github.com/neo4j/mcp/internal/server"
	"go.uber.org/mock/gomock"
)

func TestToolRegister(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDB := mocks.NewMockService(ctrl)
	dummyLogger := logger.New("info", "text", io.Discard) // Create a dummy logger

	t.Run("verifies expected tools are registered", func(t *testing.T) {
		cfg := &config.Config{
			URI:      "bolt://test-host:7687",
			Username: "neo4j",
			Password: "password",
			Database: "neo4j",
		}
		s := server.NewNeo4jMCPServer("test-version", cfg, mockDB, dummyLogger) // Pass dummyLogger

		// Expected tools that should be registered
		// update this number when a tool is added or removed.
		expectedTotalToolsCount := 4

		// Register tools
		err := s.RegisterTools()
		if err != nil {
			t.Fatalf("RegisterTools() failed: %v", err)
		}
		registeredTools := len(s.MCPServer.ListTools())

		if expectedTotalToolsCount != registeredTools {
			t.Errorf("Expected %d tools, but test configuration shows %d", expectedTotalToolsCount, registeredTools)
		}
	})

	t.Run("should register only readonly tools when readonly", func(t *testing.T) {
		cfg := &config.Config{
			URI:      "bolt://test-host:7687",
			Username: "neo4j",
			Password: "password",
			Database: "neo4j",
			ReadOnly: "true",
		}
		s := server.NewNeo4jMCPServer("test-version", cfg, mockDB, dummyLogger) // Pass dummyLogger

		// Expected tools that should be registered
		// update this number when a tool is added or removed.
		expectedTotalToolsCount := 3

		// Register tools
		err := s.RegisterTools()
		if err != nil {
			t.Fatalf("RegisterTools() failed: %v", err)
		}
		registeredTools := len(s.MCPServer.ListTools())

		if expectedTotalToolsCount != registeredTools {
			t.Errorf("Expected %d tools, but test configuration shows %d", expectedTotalToolsCount, registeredTools)
		}
	})
	t.Run("should not register only readonly tools when readonly is set to false", func(t *testing.T) {
		cfg := &config.Config{
			URI:      "bolt://test-host:7687",
			Username: "neo4j",
			Password: "password",
			Database: "neo4j",
			ReadOnly: "false",
		}
		s := server.NewNeo4jMCPServer("test-version", cfg, mockDB, dummyLogger) // Pass dummyLogger

		// Expected tools that should be registered
		// update this number when a tool is added or removed.
		expectedTotalToolsCount := 4

		// Register tools
		err := s.RegisterTools()
		if err != nil {
			t.Fatalf("RegisterTools() failed: %v", err)
		}
		registeredTools := len(s.MCPServer.ListTools())

		if expectedTotalToolsCount != registeredTools {
			t.Errorf("Expected %d tools, but test configuration shows %d", expectedTotalToolsCount, registeredTools)
		}
	})
}
