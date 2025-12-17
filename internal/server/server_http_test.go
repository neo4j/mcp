// Keeping tests in the same package to test the HTTP server without exposing internals.
package server

import (
	"crypto/tls"
	"testing"
	"time"

	analytics "github.com/neo4j/mcp/internal/analytics/mocks"
	"github.com/neo4j/mcp/internal/config"
	db "github.com/neo4j/mcp/internal/database/mocks"
	"github.com/neo4j/mcp/internal/testutil"
	"go.uber.org/mock/gomock"
)

// TestHTTPServerPortConfiguration verifies that HTTP server port and host config is stored correctly
func TestHTTPServerPortConfiguration(t *testing.T) {
	tests := []struct {
		name     string
		httpPort string
		httpHost string
	}{
		{
			name:     "default port",
			httpHost: "localhost",
			httpPort: "80",
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

			// Verify the HTTP server config is stored correctly in the server's config
			// This tests the configuration layer without running the server
			if srv.config.HTTPHost != tt.httpHost {
				t.Errorf("HTTPHost: expected %q, got %q", tt.httpHost, srv.config.HTTPHost)
			}
			if srv.config.HTTPPort != tt.httpPort {
				t.Errorf("HTTPPort: expected %q, got %q", tt.httpPort, srv.config.HTTPPort)
			}
		})
	}
}

// TestHTTPServerTLSConfiguration verifies that TLS settings are correctly stored in server config
func TestHTTPServerTLSConfiguration(t *testing.T) {
	// Generate test certificates dynamically for TLS test
	certPath, keyPath := testutil.GenerateTestTLSCertificate(t)

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

			// Verify TLS config is stored correctly in the server's config
			// This tests the configuration layer without running the server
			if srv.config.HTTPTLSEnabled != tt.tlsEnabled {
				t.Errorf("HTTPTLSEnabled: expected %v, got %v", tt.tlsEnabled, srv.config.HTTPTLSEnabled)
			}
			if srv.config.HTTPTLSCertFile != tt.tlsCertFile {
				t.Errorf("HTTPTLSCertFile: expected %q, got %q", tt.tlsCertFile, srv.config.HTTPTLSCertFile)
			}
			if srv.config.HTTPTLSKeyFile != tt.tlsKeyFile {
				t.Errorf("HTTPTLSKeyFile: expected %q, got %q", tt.tlsKeyFile, srv.config.HTTPTLSKeyFile)
			}
		})
	}
}

// TestHTTPServerTimeoutConstants verifies timeout constants are correctly defined
// This is a simpler unit test that avoids data races by not running the actual server
func TestHTTPServerTimeoutConstants(t *testing.T) {
	// Verify timeout constants are defined with expected values
	// These constants are used in StartHTTPServer() to configure the http.Server
	tests := []struct {
		name          string
		actualValue   time.Duration
		expectedValue time.Duration
		description   string
	}{
		{
			name:          "ReadHeaderTimeout",
			actualValue:   serverHTTPReadHeaderTimeout,
			expectedValue: 5 * time.Second,
			description:   "Should be 5s to prevent Slowloris attacks",
		},
		{
			name:          "ReadTimeout",
			actualValue:   serverHTTPReadTimeout,
			expectedValue: 15 * time.Second,
			description:   "Should be 15s to prevent slow-read attacks",
		},
		{
			name:          "WriteTimeout",
			actualValue:   serverHTTPWriteTimeout,
			expectedValue: 60 * time.Second,
			description:   "Should be 60s to allow complex Neo4j queries",
		},
		{
			name:          "IdleTimeout",
			actualValue:   serverHTTPIdleTimeout,
			expectedValue: 120 * time.Second,
			description:   "Should be 120s for keep-alive connection reuse",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.actualValue != tt.expectedValue {
				t.Errorf("%s: expected %v, got %v (%s)", tt.name, tt.expectedValue, tt.actualValue, tt.description)
			}
		})
	}
}

// TestBuildTLSConfig verifies the TLS configuration building logic without starting a server
func TestBuildTLSConfig(t *testing.T) {
	tests := []struct {
		name        string
		setupCerts  bool
		expectError bool
	}{
		{
			name:        "valid certificates",
			setupCerts:  true,
			expectError: false,
		},
		{
			name:        "missing certificate files",
			setupCerts:  false,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var certPath, keyPath string
			if tt.setupCerts {
				certPath, keyPath = testutil.GenerateTestTLSCertificate(t)
			} else {
				certPath = "/nonexistent/cert.pem"
				keyPath = "/nonexistent/key.pem"
			}

			cfg := &config.Config{
				HTTPTLSCertFile: certPath,
				HTTPTLSKeyFile:  keyPath,
			}

			srv := &Neo4jMCPServer{config: cfg}

			tlsConfig, err := srv.buildTLSConfig()

			if tt.expectError {
				if err == nil {
					t.Error("Expected error for missing certificate files, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("buildTLSConfig() failed: %v", err)
			}

			// Verify TLS configuration
			if tlsConfig.MinVersion != tls.VersionTLS12 {
				t.Errorf("Expected MinVersion TLS 1.2 (0x0303), got 0x%x", tlsConfig.MinVersion)
			}

			// Verify cipher suites are using Go defaults (nil)
			if tlsConfig.CipherSuites != nil {
				t.Error("Expected CipherSuites to be nil (using Go defaults)")
			}
		})
	}
}
