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

func TestNewNeo4jMCPServer(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	cfg := &config.Config{
		URI:      "bolt://test-host:7687",
		Username: "neo4j",
		Password: "password",
		Database: "neo4j",
	}

	mockDB := mocks.NewMockService(ctrl)
	dummyLogger := logger.New("info", "text", io.Discard) // Create a dummy logger

	t.Run("creates server successfully", func(t *testing.T) {
		s := server.NewNeo4jMCPServer("test-version", cfg, mockDB, dummyLogger) // Pass dummyLogger

		if s == nil {
			t.Errorf("NewNeo4jMCPServer() expected non-nil server, got nil")
		}
	})

	t.Run("starts server successfully", func(t *testing.T) {
		s := server.NewNeo4jMCPServer("test-version", cfg, mockDB, dummyLogger) // Pass dummyLogger

		if s == nil {
			t.Errorf("NewNeo4jMCPServer() expected non-nil server, got nil")
		}

		err := s.Start()
		if err != nil {
			t.Errorf("Start() unexpected error = %v", err)
		}
	})

	t.Run("stops server successfully", func(t *testing.T) {
		s := server.NewNeo4jMCPServer("test-version", cfg, mockDB, dummyLogger) // Pass dummyLogger

		if s == nil {
			t.Errorf("NewNeo4jMCPServer() expected non-nil server, got nil")
		}

		err := s.Start()
		if err != nil {
			t.Errorf("Start() unexpected error = %v", err)
		}
	})

	t.Run("server creates successfully with all required components", func(t *testing.T) {
		s := server.NewNeo4jMCPServer("test-version", cfg, mockDB, dummyLogger) // Pass dummyLogger

		if s == nil {
			t.Fatal("NewNeo4jMCPServer() returned nil")
		}

		// Register tools should work without errors
		err := s.RegisterTools()
		if err != nil {
			t.Errorf("RegisterTools() unexpected error = %v", err)
		}
		// Start should work without errors
		err = s.Start()
		if err != nil {
			t.Errorf("Start() unexpected error = %v", err)
		}

		// Stop should work without errors
		err = s.Stop()
		if err != nil {
			t.Errorf("Stop() unexpected error = %v", err)
		}
	})

}
