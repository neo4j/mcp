package server_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/neo4j/mcp/internal/config"
	"github.com/neo4j/mcp/internal/server"
)

func TestNewNeo4jMCPServer(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *config.Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid config",
			cfg: &config.Config{
				URI:      "bolt://localhost:7687",
				Username: "neo4j",
				Password: "password",
				Database: "neo4j",
			},
			wantErr: false,
		},
		{
			name: "invalid URI format",
			cfg: &config.Config{
				URI:      "invalid://uri:format",
				Username: "neo4j",
				Password: "password",
				Database: "neo4j",
			},
			wantErr: true,
			errMsg:  "failed to create Neo4j driver",
		},
		{
			name: "empty URI",
			cfg: &config.Config{
				URI:      "",
				Username: "neo4j",
				Password: "password",
				Database: "neo4j",
			},
			wantErr: true,
			errMsg:  "Neo4j URI is required but was empty",
		},
		{
			name: "empty username",
			cfg: &config.Config{
				URI:      "bolt://localhost:7687",
				Username: "",
				Password: "password",
				Database: "neo4j",
			},
			wantErr: true,
			errMsg:  "Neo4j username is required but was empty",
		},
		{
			name: "empty password",
			cfg: &config.Config{
				URI:      "bolt://localhost:7687",
				Username: "neo4j",
				Password: "",
				Database: "neo4j",
			},
			wantErr: true,
			errMsg:  "Neo4j password is required but was empty",
		},
		{
			name: "empty database",
			cfg: &config.Config{
				URI:      "bolt://localhost:7687",
				Username: "neo4j",
				Password: "password",
				Database: "",
			},
			wantErr: true,
			errMsg:  "Neo4j database name is required but was empty",
		},
		{
			name: "nil config",
			cfg:  nil,
			wantErr: true,
			errMsg:  "configuration is required but was nil",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server, err := server.NewNeo4jMCPServer(tt.cfg)

			if tt.wantErr {
				if err == nil {
					t.Errorf("NewNeo4jMCPServer() expected error but got none")
					return
				}
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("NewNeo4jMCPServer() error = %v, want error containing %v", err, tt.errMsg)
				}
				if server != nil {
					t.Errorf("NewNeo4jMCPServer() expected nil server on error, got %v", server)
				}
				return
			}

			if err != nil {
				t.Errorf("NewNeo4jMCPServer() unexpected error = %v", err)
				return
			}

			if server == nil {
				t.Errorf("NewNeo4jMCPServer() expected non-nil server, got nil")
			}
		})
	}
}

func TestNeo4jMCPServer_RegisterTools(t *testing.T) {
	cfg := &config.Config{
		URI:      "bolt://localhost:7687",
		Username: "neo4j",
		Password: "password",
		Database: "neo4j",
	}

	s, err := server.NewNeo4jMCPServer(cfg)
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	err = s.RegisterTools()
	if err != nil {
		t.Errorf("RegisterTools() unexpected error = %v", err)
	}
}

func TestNeo4jMCPServer_Start_ConnectionFailure(t *testing.T) {
	// Test with invalid connection that will fail verification quickly
	cfg := &config.Config{
		URI:      "bolt://127.0.0.1:9999", // Use localhost with invalid port for faster failure
		Username: "neo4j",
		Password: "wrongpassword",
		Database: "neo4j",
	}

	s, err := server.NewNeo4jMCPServer(cfg)
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	err = s.Start(ctx)

	if err == nil {
		t.Error("Start() expected error for invalid connection but got none")
		return
	}

	if !strings.Contains(err.Error(), "failed to verify database connectivity") {
		t.Errorf("Start() error = %v, want error containing 'failed to verify database connectivity'", err)
	}
}

func TestNeo4jMCPServer_Start_WithCanceledContext(t *testing.T) {
	cfg := &config.Config{
		URI:      "bolt://127.0.0.1:9999", // Use localhost with invalid port for faster failure
		Username: "neo4j",
		Password: "wrongpassword",
		Database: "neo4j",
	}

	s, err := server.NewNeo4jMCPServer(cfg)
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	// Create a context that's already canceled
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err = s.Start(ctx)

	if err == nil {
		t.Error("Start() expected error with canceled context but got none")
		return
	}

	// Should fail with context cancellation or connectivity error
	expectedErrors := []string{
		"failed to verify database connectivity",
		"context canceled",
	}

	containsExpectedError := false
	for _, expectedErr := range expectedErrors {
		if strings.Contains(err.Error(), expectedErr) {
			containsExpectedError = true
			break
		}
	}

	if !containsExpectedError {
		t.Errorf("Start() error = %v, want error containing one of %v", err, expectedErrors)
	}
}

func TestNeo4jMCPServer_Stop(t *testing.T) {
	cfg := &config.Config{
		URI:      "bolt://localhost:7687",
		Username: "neo4j",
		Password: "password",
		Database: "neo4j",
	}

	s, err := server.NewNeo4jMCPServer(cfg)
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	ctx := context.Background()
	err = s.Stop(ctx)

	// Stop should not return an error even if the connection was never established
	if err != nil {
		t.Errorf("Stop() unexpected error = %v", err)
	}
}

func TestNeo4jMCPServer_StopWithTimeout(t *testing.T) {
	cfg := &config.Config{
		URI:      "bolt://localhost:7687",
		Username: "neo4j",
		Password: "password",
		Database: "neo4j",
	}

	s, err := server.NewNeo4jMCPServer(cfg)
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	// Create a context with timeout
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately to test timeout behavior

	err = s.Stop(ctx)

	// Should handle context cancellation gracefully
	if err != nil && !errors.Is(err, context.Canceled) {
		t.Errorf("Stop() with cancelled context error = %v, want context.Canceled or no error", err)
	}
}
