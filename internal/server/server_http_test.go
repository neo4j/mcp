// Keeping tests in the same package to test the HTTP server without exposing internals.
package server

import (
	"context"
	"fmt"
	"testing"
	"time"

	analytics "github.com/neo4j/mcp/internal/analytics/mocks"
	"github.com/neo4j/mcp/internal/config"
	db "github.com/neo4j/mcp/internal/database/mocks"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"go.uber.org/mock/gomock"
)

// TestHTTPServerPortConfiguration verifies that the HTTP server uses the configured port
func TestHTTPServerPortConfiguration(t *testing.T) {
	tests := []struct {
		name     string
		httpPort string
		httpHost string
	}{
		{
			name:     "default port",
			httpHost: "localhost",
			httpPort: "8080",
		},
		{
			name:     "custom port",
			httpHost: "127.0.0.1",
			httpPort: "9999",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			cfg := &config.Config{
				URI:           "bolt://test-host:7687",
				Username:      "test-username",
				Password:      "test-password",
				Database:      "neo4j",
				TransportMode: config.TransportModeHTTP,
				HTTPHost:      tt.httpHost,
				HTTPPort:      tt.httpPort,
			}

			// Setup mocks for server initialization
			mockDB := db.NewMockService(ctrl)
			mockDB.EXPECT().VerifyConnectivity(gomock.Any()).Return(nil)
			mockDB.EXPECT().ExecuteReadQuery(gomock.Any(), "RETURN 1 as first", gomock.Any()).Return([]*neo4j.Record{
				{Keys: []string{"first"}, Values: []any{int64(1)}},
			}, nil)
			checkApocMetaSchemaQuery := "SHOW PROCEDURES YIELD name WHERE name = 'apoc.meta.schema' RETURN count(name) > 0 AS apocMetaSchemaAvailable"
			mockDB.EXPECT().ExecuteReadQuery(gomock.Any(), checkApocMetaSchemaQuery, gomock.Any()).Return([]*neo4j.Record{
				{Keys: []string{"apocMetaSchemaAvailable"}, Values: []any{true}},
			}, nil)
			gdsVersionQuery := "RETURN gds.version() as gdsVersion"
			mockDB.EXPECT().ExecuteReadQuery(gomock.Any(), gdsVersionQuery, gomock.Any()).Return([]*neo4j.Record{
				{Keys: []string{"gdsVersion"}, Values: []any{"2.22.0"}},
			}, nil)
			// Mock for emitStartupEvent
			mockDB.EXPECT().ExecuteReadQuery(gomock.Any(), "CALL dbms.components()", gomock.Any()).Return([]*neo4j.Record{}, nil)

			analyticsService := analytics.NewMockService(ctrl)
			analyticsService.EXPECT().NewStartupEvent(gomock.Any()).AnyTimes()
			analyticsService.EXPECT().EmitEvent(gomock.Any()).AnyTimes()

			srv := NewNeo4jMCPServer("test-version", cfg, mockDB, analyticsService)
			if srv == nil {
				t.Fatal("Expected non-nil server")
			}

			// Start server briefly to initialize httpServer
			errChan := make(chan error, 1)
			go func() {
				errChan <- srv.Start()
			}()

			// Wait for server initialization
			time.Sleep(100 * time.Millisecond)

			// Verify the HTTP server is configured with the expected address
			if srv.httpServer == nil {
				t.Fatal("httpServer should be initialized")
			}

			expectedAddr := fmt.Sprintf("%s:%s", tt.httpHost, tt.httpPort)
			if srv.httpServer.Addr != expectedAddr {
				t.Errorf("Expected server address %s, got %s", expectedAddr, srv.httpServer.Addr)
			}

			// Cleanup
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			if err := srv.Stop(ctx); err != nil {
				t.Errorf("Failed to stop server: %v", err)
			}
		})
	}
}

// TestHTTPServerTimeoutValues verifies the actual http.Server timeout configuration
func TestHTTPServerTimeoutValues(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Setup config with HTTP transport
	cfg := &config.Config{
		URI:           "bolt://test-host:7687",
		Username:      "test-username",
		Password:      "test-password",
		Database:      "neo4j",
		TransportMode: config.TransportModeHTTP,
		HTTPHost:      "127.0.0.1",
		HTTPPort:      "0", // Use port 0 to get a random available port
	}

	// Setup mocks for server initialization
	mockDB := db.NewMockService(ctrl)
	mockDB.EXPECT().VerifyConnectivity(gomock.Any()).Return(nil)
	mockDB.EXPECT().ExecuteReadQuery(gomock.Any(), "RETURN 1 as first", gomock.Any()).Return([]*neo4j.Record{
		{Keys: []string{"first"}, Values: []any{int64(1)}},
	}, nil)
	checkApocMetaSchemaQuery := "SHOW PROCEDURES YIELD name WHERE name = 'apoc.meta.schema' RETURN count(name) > 0 AS apocMetaSchemaAvailable"
	mockDB.EXPECT().ExecuteReadQuery(gomock.Any(), checkApocMetaSchemaQuery, gomock.Any()).Return([]*neo4j.Record{
		{Keys: []string{"apocMetaSchemaAvailable"}, Values: []any{true}},
	}, nil)
	gdsVersionQuery := "RETURN gds.version() as gdsVersion"
	mockDB.EXPECT().ExecuteReadQuery(gomock.Any(), gdsVersionQuery, gomock.Any()).Return([]*neo4j.Record{
		{Keys: []string{"gdsVersion"}, Values: []any{"2.22.0"}},
	}, nil)
	// Mock for emitStartupEvent
	mockDB.EXPECT().ExecuteReadQuery(gomock.Any(), "CALL dbms.components()", gomock.Any()).Return([]*neo4j.Record{}, nil)

	analyticsService := analytics.NewMockService(ctrl)
	analyticsService.EXPECT().NewStartupEvent(gomock.Any()).AnyTimes()
	analyticsService.EXPECT().EmitEvent(gomock.Any()).AnyTimes()

	srv := NewNeo4jMCPServer("test-version", cfg, mockDB, analyticsService)

	// Start server in background (it will block on ListenAndServe)
	errChan := make(chan error, 1)
	go func() {
		errChan <- srv.Start()
	}()

	// Wait briefly for server to initialize (httpServer is set before ListenAndServe blocks)
	// Relying on timing here is not ideal, we should consider adding a ready channel in the server in the future or moving this to an integration test.
	time.Sleep(100 * time.Millisecond)

	// Now we can access the private httpServer field since we're in package server
	if srv.httpServer == nil {
		t.Fatal("httpServer should be initialized")
	}

	// Verify timeout values match what's in server.go:192-195
	expectedTimeouts := struct {
		Read       time.Duration
		Write      time.Duration
		Idle       time.Duration
		ReadHeader time.Duration
	}{
		Read:       10 * time.Second,
		Write:      30 * time.Second,
		Idle:       60 * time.Second,
		ReadHeader: 5 * time.Second,
	}

	if srv.httpServer.ReadTimeout != expectedTimeouts.Read {
		t.Errorf("ReadTimeout: expected %v, got %v", expectedTimeouts.Read, srv.httpServer.ReadTimeout)
	}
	if srv.httpServer.WriteTimeout != expectedTimeouts.Write {
		t.Errorf("WriteTimeout: expected %v, got %v", expectedTimeouts.Write, srv.httpServer.WriteTimeout)
	}
	if srv.httpServer.IdleTimeout != expectedTimeouts.Idle {
		t.Errorf("IdleTimeout: expected %v, got %v", expectedTimeouts.Idle, srv.httpServer.IdleTimeout)
	}
	if srv.httpServer.ReadHeaderTimeout != expectedTimeouts.ReadHeader {
		t.Errorf("ReadHeaderTimeout: expected %v, got %v", expectedTimeouts.ReadHeader, srv.httpServer.ReadHeaderTimeout)
	}

	// Cleanup - stop the server
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := srv.Stop(ctx); err != nil {
		t.Errorf("Failed to stop server: %v", err)
	}

	// Verify server stopped (check error from Start goroutine)
	select {
	case err := <-errChan:
		// Server stopped, error should be about http.ErrServerClosed or nil
		if err != nil {
			t.Logf("Server stopped with: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Error("Server did not stop within timeout")
	}
}
