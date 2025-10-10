package server_test

import (
	"testing"

	"github.com/neo4j/mcp/internal/config"
	"github.com/neo4j/mcp/internal/database/mocks"
	"github.com/neo4j/mcp/internal/server"
	"go.uber.org/mock/gomock"
)

func TestNewNeo4jMCPServer(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	cfg := &config.Config{
		URI:      "bolt://localhost:7687",
		Username: "neo4j",
		Password: "password",
		Database: "neo4j",
	}

	mockDB := mocks.NewMockDatabaseService(ctrl)

	t.Run("creates server successfully", func(t *testing.T) {
		s := server.NewNeo4jMCPServer("test-version", cfg, mockDB)

		if s == nil {
			t.Errorf("NewNeo4jMCPServer() expected non-nil server, got nil")
		}
	})
}

func TestNeo4jMCPServer_RegisterTools(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	cfg := &config.Config{
		URI:      "bolt://localhost:7687",
		Username: "neo4j",
		Password: "password",
		Database: "neo4j",
	}

	mockDB := mocks.NewMockDatabaseService(ctrl)
	s := server.NewNeo4jMCPServer("test-version", cfg, mockDB)

	err := s.RegisterTools()
	if err != nil {
		t.Errorf("RegisterTools() unexpected error = %v", err)
	}
}

func TestNeo4jMCPServer_Stop(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	cfg := &config.Config{
		URI:      "bolt://localhost:7687",
		Username: "neo4j",
		Password: "password",
		Database: "neo4j",
	}

	mockDB := mocks.NewMockDatabaseService(ctrl)
	s := server.NewNeo4jMCPServer("test-version", cfg, mockDB)

	err := s.Stop()

	if err != nil {
		t.Errorf("Stop() unexpected error = %v", err)
	}
}
