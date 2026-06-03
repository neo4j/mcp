// Copyright (c) "Neo4j"
// Neo4j Sweden AB [http://neo4j.com]

// Package server_test contains tests for the HTTP mode lifecycle of the Neo4j MCP server.
// This file specifically tests the differences between HTTP and STDIO transport modes,
// with focus on the delayed initialization pattern used in HTTP mode.
//
// Key differences tested:
// - HTTP mode: Verification and tool registration are delayed until the first client initializes
// - STDIO mode: Verification and tool registration happen immediately during Start()
//
// The HTTP mode uses hooks to defer the initialization process because:
// - Database credentials are provided per-request via Basic Auth headers or Bearer Token
// - No credentials are available at server startup time
// - The server must start immediately to serve HTTP requests
package server_test

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"sort"
	"strconv"
	"testing"
	"time"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/client/transport"
	"github.com/mark3labs/mcp-go/mcp"
	analytics "github.com/neo4j/mcp/internal/analytics/mocks"
	"github.com/neo4j/mcp/internal/config"
	db "github.com/neo4j/mcp/internal/database/mocks"
	server "github.com/neo4j/mcp/internal/server"
	"github.com/neo4j/neo4j-go-driver/v6/neo4j"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

// TestNeo4jMCPServerHTTPMode tests the HTTP mode lifecycle where initialization is delayed
// until the first client performs an initialize request via hooks
func TestNeo4jMCPServerHTTPMode(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Find a free port for the test
	port, err := getFreePort()
	if err != nil {
		t.Fatalf("Failed to find free port: %v", err)
	}

	// Base configuration for HTTP mode
	cfg := &config.Config{
		URI:           "bolt://test-host:7687",
		Database:      "neo4j",
		TransportMode: config.TransportModeHTTP,
		Tools:         config.AvailableTools,
		HTTPHost:      "127.0.0.1",
		HTTPPort:      strconv.Itoa(port),
	}
	uri := fmt.Sprintf("http://%s:%s/db/neo4j/mcp", cfg.HTTPHost, cfg.HTTPPort)

	analyticsService := analytics.NewMockService(ctrl)
	analyticsService.EXPECT().EmitEvent(gomock.Any()).AnyTimes()
	analyticsService.EXPECT().NewStartupEvent(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	analyticsService.EXPECT().IsEnabled().AnyTimes().Return(true)
	analyticsService.EXPECT().NewConnectionInitializedEvent(gomock.Any()).AnyTimes()

	t.Run("HTTP mode starts without verification and registers hook", func(t *testing.T) {
		// In HTTP mode, no DB verification should happen at startup
		mockDB := db.NewMockService(ctrl)
		// No expectations for DB calls during Start() in HTTP mode
		s, errChan := createHTTPServer(t, cfg, mockDB, analyticsService)

		assertNoCloseOrStopError(t, s, errChan)
	})

	t.Run("Server triggers verification on first initialize request", func(t *testing.T) {
		mockDB := db.NewMockService(ctrl)
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
		mockDB.EXPECT().ExecuteReadQuery(gomock.Any(), gdsVersionQuery, gomock.Any()).Times(1).Return([]*neo4j.Record{
			{
				Keys: []string{"gdsVersion"},
				Values: []any{
					string("2.22.0"),
				},
			},
		}, nil)

		mockDB.EXPECT().ExecuteReadQuery(gomock.Any(), "CALL dbms.components()", gomock.Any()).Times(1)

		// In HTTP mode, NO database operations should happen during Start()
		// The hook is registered but not executed until a real client request
		s, errChan := createHTTPServer(t, cfg, mockDB, analyticsService)
		// Signal that server is ready to accept requests

		mcpClient := createStreamableHTTPClient(uri, defaultHeaders())
		_, err := mcpClient.Initialize(context.Background(), mcp.InitializeRequest{})
		if err != nil {
			t.Fatalf("error while initialize request: %v", err)
		}
		assertNoCloseOrStopError(t, s, errChan)
	})

	t.Run("Server handles database connectivity errors gracefully", func(t *testing.T) {
		mockDB := db.NewMockService(ctrl)
		// in HTTP the serve should keep running even if the connectivity check fails.
		// This is because the client can be misconfigured with invalid credentials
		// and it should not affect the experience to other clients/users with correct information.

		mockDB.EXPECT().ExecuteReadQuery(gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return(nil, fmt.Errorf("connection error"))
		// In HTTP mode, no database calls happen during Start()
		// The hook will handle errors when actually triggered by a client request
		s, errChan := createHTTPServer(t, cfg, mockDB, analyticsService)

		mcpClient := createStreamableHTTPClient(uri, defaultHeaders())
		// initialize should fail, while the server should keep working fine.
		_, err := mcpClient.Initialize(context.Background(), mcp.InitializeRequest{})
		if err == nil {
			t.Fatalf("error while initialize request: %v", err)
		}
		assert.ErrorContains(t, err, "impossible to verify connectivity with the Neo4j instance: connection error")
		assertNoCloseOrStopError(t, s, errChan)
	})

	t.Run("server creates successfully with all required components", func(t *testing.T) {
		mockDB := db.NewMockService(ctrl)
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
		mockDB.EXPECT().ExecuteReadQuery(gomock.Any(), gdsVersionQuery, gomock.Any()).Times(1).Return([]*neo4j.Record{
			{
				Keys: []string{"gdsVersion"},
				Values: []any{
					string("2.22.0"),
				},
			},
		}, nil)

		mockDB.EXPECT().ExecuteReadQuery(gomock.Any(), "CALL dbms.components()", gomock.Any()).Times(1)

		// In HTTP mode, NO database operations should happen during Start()
		// The hook is registered but not executed until a real client request
		s, errChan := createHTTPServer(t, cfg, mockDB, analyticsService)

		mcpClient := createStreamableHTTPClient(uri, defaultHeaders())
		_, err := mcpClient.Initialize(context.Background(), mcp.InitializeRequest{})
		if err != nil {
			t.Fatalf("error while initialize request: %v", err)
		}

		toolNames := make([]string, 0, len(s.MCPServer.ListTools()))
		for _, tool := range s.MCPServer.ListTools() {
			toolNames = append(toolNames, tool.Tool.Name)
		}
		assert.Contains(t, toolNames, "list-gds-procedures")

		assertNoCloseOrStopError(t, s, errChan)

	})
}

// TestNeo4jMCPServerHTTPModeToolsFilter tests the per-request ToolFilter behaviour
// driven by the X-Neo4j-MCP-Readonly and X-Neo4j-MCP-Tools headers.
func TestNeo4jMCPServerHTTPModeToolsFilter(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	port, err := getFreePort()
	if err != nil {
		t.Fatalf("Failed to find free port: %v", err)
	}

	cfg := &config.Config{
		URI:           "bolt://test-host:7687",
		Database:      "neo4j",
		TransportMode: config.TransportModeHTTP,
		Tools:         config.AvailableTools,
		HTTPHost:      "127.0.0.1",
		HTTPPort:      strconv.Itoa(port),
	}
	uri := fmt.Sprintf("http://%s:%s/db/neo4j/mcp", cfg.HTTPHost, cfg.HTTPPort)

	analyticsService := analytics.NewMockService(ctrl)
	analyticsService.EXPECT().EmitEvent(gomock.Any()).AnyTimes()
	analyticsService.EXPECT().NewStartupEvent(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	analyticsService.EXPECT().IsEnabled().AnyTimes().Return(true)
	analyticsService.EXPECT().NewConnectionInitializedEvent(gomock.Any()).AnyTimes()

	mockDB := db.NewMockService(ctrl)
	mockDB.EXPECT().ExecuteReadQuery(gomock.Any(), "RETURN 1 as first", gomock.Any()).
		AnyTimes().Return([]*neo4j.Record{{Keys: []string{"first"}, Values: []any{int64(1)}}}, nil)
	mockDB.EXPECT().ExecuteReadQuery(gomock.Any(), "SHOW PROCEDURES YIELD name WHERE name = 'apoc.meta.schema' RETURN count(name) > 0 AS apocMetaSchemaAvailable", gomock.Any()).
		AnyTimes().Return([]*neo4j.Record{{Keys: []string{"apocMetaSchemaAvailable"}, Values: []any{bool(true)}}}, nil)
	mockDB.EXPECT().ExecuteReadQuery(gomock.Any(), "RETURN gds.version() as gdsVersion", gomock.Any()).
		AnyTimes().Return([]*neo4j.Record{{Keys: []string{"gdsVersion"}, Values: []any{string("2.22.0")}}}, nil)
	mockDB.EXPECT().ExecuteReadQuery(gomock.Any(), "CALL dbms.components()", gomock.Any()).
		AnyTimes().Return(nil, nil)

	tests := []struct {
		name          string
		extraHeaders  map[string]string
		wantErr       bool
		wantToolNames []string
	}{
		{
			name:          "All tools returned when X-Neo4j-MCP-Readonly is false",
			extraHeaders:  map[string]string{"X-Neo4j-MCP-Readonly": "false"},
			wantToolNames: []string{"get-schema", "list-gds-procedures", "read-cypher", "write-cypher"},
		},
		{
			name:          "All tools returned when X-Neo4j-MCP-Readonly is False (mixed case)",
			extraHeaders:  map[string]string{"X-Neo4j-MCP-Readonly": "False"},
			wantToolNames: []string{"get-schema", "list-gds-procedures", "read-cypher", "write-cypher"},
		},
		{
			name:          "Only read-only tools returned when X-Neo4j-MCP-Readonly is true",
			extraHeaders:  map[string]string{"X-Neo4j-MCP-Readonly": "true"},
			wantToolNames: []string{"get-schema", "list-gds-procedures", "read-cypher"},
		},
		{
			name:         "Error when X-Neo4j-MCP-Readonly contains an invalid value",
			extraHeaders: map[string]string{"X-Neo4j-MCP-Readonly": "invalid"},
			wantErr:      true,
		},
		{
			name:          "Single tool filter via X-Neo4j-MCP-Tools",
			extraHeaders:  map[string]string{"X-Neo4j-MCP-Tools": "read-cypher"},
			wantToolNames: []string{"read-cypher"},
		},
		{
			name:          "Comma-separated tools via X-Neo4j-MCP-Tools",
			extraHeaders:  map[string]string{"X-Neo4j-MCP-Tools": "read-cypher, write-cypher"},
			wantToolNames: []string{"read-cypher", "write-cypher"},
		},
		{
			// mcp-go wraps 400 errors: "server returned 4xx for initialize POST, likely a legacy SSE server"
			name:         "Error when X-Neo4j-MCP-Tools contains an invalid tool name",
			extraHeaders: map[string]string{"X-Neo4j-MCP-Tools": "batman-tool"},
			wantErr:      true,
		},
		{
			// mcp-go wraps 400 errors: "server returned 4xx for initialize POST, likely a legacy SSE server"
			name:         "Error when X-Neo4j-MCP-Tools contains an mixed valid tool names",
			extraHeaders: map[string]string{"X-Neo4j-MCP-Tools": "read-cypher, batman-tool"},
			wantErr:      true,
		},
		{
			// mcp-go wraps 400 errors: "server returned 4xx for initialize POST, likely a legacy SSE server"
			name:         "Error when X-Neo4j-MCP-Tools is set as empty \"\" string",
			extraHeaders: map[string]string{"X-Neo4j-MCP-Tools": ""},
			wantErr:      true,
		},
		{
			name: "X-Neo4j-MCP-Tools and X-Neo4j-MCP-Readonly applied as intersection",
			extraHeaders: map[string]string{
				"X-Neo4j-MCP-Tools":    "read-cypher, write-cypher",
				"X-Neo4j-MCP-Readonly": "true",
			},
			// write-cypher is not read-only, so it is excluded despite being in the tools list
			wantToolNames: []string{"read-cypher"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s, errChan := createHTTPServer(t, cfg, mockDB, analyticsService)

			headers := defaultHeaders()
			for k, v := range tc.extraHeaders {
				headers[k] = v
			}
			mcpClient := createStreamableHTTPClient(uri, headers)

			_, err := mcpClient.Initialize(context.Background(), mcp.InitializeRequest{})
			if tc.wantErr {
				assert.Error(t, err)
				assertNoCloseOrStopError(t, s, errChan)
				return
			}
			if err != nil {
				t.Fatalf("expected initialize to succeed, got: %v", err)
			}

			listToolsResponse, err := mcpClient.ListTools(context.Background(), mcp.ListToolsRequest{})
			if err != nil {
				t.Fatalf("failed to list tools: %v", err)
			}
			toolNames := toolNamesFrom(listToolsResponse.Tools)
			sort.Strings(toolNames)
			assert.Equal(t, tc.wantToolNames, toolNames)

			assertNoCloseOrStopError(t, s, errChan)
		})
	}
}

func createHTTPServer(t *testing.T, cfg *config.Config, mockDB *db.MockService, analyticsService *analytics.MockService) (*server.Neo4jMCPServer, chan error) {
	s := server.NewNeo4jMCPServer("test-version", cfg, mockDB, analyticsService)

	if s == nil {
		t.Fatal("NewNeo4jMCPServer() returned nil")
	}

	// Start HTTP server in goroutine since it's blocking
	errChan := make(chan error, 1)
	go func() {
		err := s.Start()
		if err != nil {
			errChan <- err
		}
	}()
	// wait for HttpServerReady to be closed
	for range s.HTTPServerReady { //nolint:all // Waiting for channel to close
	}

	// Give the server a moment to actually bind to the port
	// The HTTPServerReady channel closes before ListenAndServe() is called
	time.Sleep(100 * time.Millisecond)

	return s, errChan
}

func toolNamesFrom(tools []mcp.Tool) []string {
	names := make([]string, len(tools))
	for i, tool := range tools {
		names[i] = tool.Name
	}
	return names
}

func assertNoCloseOrStopError(t *testing.T, s *server.Neo4jMCPServer, errChan chan error) {
	// Stop the server
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err := s.Stop(ctx)
	if err != nil {
		t.Errorf("Stop() unexpected error = %v", err)
	}

	// Check if there were any startup errors
	select {
	case err := <-errChan:
		t.Errorf("Start() unexpected error = %v", err)
	default:
		// No error, which is expected
	}
}

// defaultHeaders returns the base request headers used by most test clients.
func defaultHeaders() map[string]string {
	return map[string]string{
		"Authorization":   "Basic bmVvNGo6cGFzc3dvcmQ=",
		"X-Neo4j-MCP-URI": "bolt://test-host:7687",
	}
}

func createStreamableHTTPClient(url string, headers map[string]string) *client.Client {
	httpTransport, err := transport.NewStreamableHTTP(url,
		transport.WithHTTPTimeout(30*time.Second),
		transport.WithHTTPHeaders(headers),
		transport.WithHTTPBasicClient(&http.Client{}),
	)
	if err != nil {
		log.Fatalf("Failed to create StreamableHTTP transport: %v", err)
	}
	c := client.NewClient(httpTransport)
	return c
}

// getFreePort finds and returns an available port
func getFreePort() (int, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer listener.Close()
	addr := listener.Addr().(*net.TCPAddr)
	return addr.Port, nil
}
