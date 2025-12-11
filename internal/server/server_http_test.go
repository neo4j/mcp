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
			// Note: In HTTP mode, verification is skipped (no DB queries at startup)
			mockDB := db.NewMockService(ctrl)

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

			// Wait for server to signal that httpServer is initialized
			<-srv.httpServerReady

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

// TestHTTPServerTLSConfiguration verifies that the HTTP server correctly uses TLS settings
func TestHTTPServerTLSConfiguration(t *testing.T) {
	// Generate test certificates dynamically for TLS test
	certPath, keyPath := config.GenerateTestTLSCertificate(t)

	tests := []struct {
		name           string
		tlsEnabled     bool
		tlsCertFile    string
		tlsKeyFile     string
		expectTLSSetup bool
	}{
		{
			name:           "TLS enabled with cert and key",
			tlsEnabled:     true,
			tlsCertFile:    certPath,
			tlsKeyFile:     keyPath,
			expectTLSSetup: true,
		},
		{
			name:           "TLS disabled",
			tlsEnabled:     false,
			tlsCertFile:    "",
			tlsKeyFile:     "",
			expectTLSSetup: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			cfg := &config.Config{
				URI:             "bolt://test-host:7687",
				Username:        "test-username",
				Password:        "test-password",
				Database:        "neo4j",
				TransportMode:   config.TransportModeHTTP,
				HTTPHost:        "127.0.0.1",
				HTTPPort:        "0", // Use port 0 to get a random available port
				HTTPTLSEnabled:  tt.tlsEnabled,
				HTTPTLSCertFile: tt.tlsCertFile,
				HTTPTLSKeyFile:  tt.tlsKeyFile,
			}

			// Setup mocks for server initialization
			// Note: In HTTP mode, verification is skipped (no DB queries at startup)
			mockDB := db.NewMockService(ctrl)

			analyticsService := analytics.NewMockService(ctrl)
			analyticsService.EXPECT().NewStartupEvent(gomock.Any()).AnyTimes()
			analyticsService.EXPECT().EmitEvent(gomock.Any()).AnyTimes()

			srv := NewNeo4jMCPServer("test-version", cfg, mockDB, analyticsService)
			if srv == nil {
				t.Fatal("Expected non-nil server")
			}

			// Verify config is stored correctly
			if srv.config.HTTPTLSEnabled != tt.tlsEnabled {
				t.Errorf("Expected HTTPTLSEnabled %v, got %v", tt.tlsEnabled, srv.config.HTTPTLSEnabled)
			}

			// Start server briefly to initialize httpServer
			errChan := make(chan error, 1)
			go func() {
				errChan <- srv.Start()
			}()

			// Wait for server initialization
			time.Sleep(100 * time.Millisecond)

			// Verify the HTTP server is initialized
			if srv.httpServer == nil {
				t.Fatal("httpServer should be initialized")
			}

			// Cleanup
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			if err := srv.Stop(ctx); err != nil {
				t.Errorf("Failed to stop server: %v", err)
			}

			// Verify server stopped
			select {
			case err := <-errChan:
				// Server stopped, error should be about http.ErrServerClosed or nil
				if err != nil {
					t.Logf("Server stopped with: %v", err)
				}
			case <-time.After(3 * time.Second):
				t.Error("Server did not stop within timeout")
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
	// Note: In HTTP mode, verification is skipped (no DB queries at startup)
	mockDB := db.NewMockService(ctrl)

	analyticsService := analytics.NewMockService(ctrl)
	analyticsService.EXPECT().NewStartupEvent(gomock.Any()).AnyTimes()
	analyticsService.EXPECT().EmitEvent(gomock.Any()).AnyTimes()

	srv := NewNeo4jMCPServer("test-version", cfg, mockDB, analyticsService)

	// Start server in background (it will block on ListenAndServe)
	errChan := make(chan error, 1)
	go func() {
		errChan <- srv.Start()
	}()

	// Wait for server to signal that httpServer is initialized
	<-srv.httpServerReady

	// Now we can safely access the httpServer field since we're in package server
	if srv.httpServer == nil {
		t.Fatal("httpServer should be initialized")
	}

	// Verify timeout values match the constants in server.go
	expectedTimeouts := struct {
		Read       time.Duration
		Write      time.Duration
		Idle       time.Duration
		ReadHeader time.Duration
	}{
		Read:       10 * time.Second,
		Write:      60 * time.Second,
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
