// Copyright (c) "Neo4j"
// Neo4j Sweden AB [http://neo4j.com]

//go:build integration

package integration

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	analytics "github.com/neo4j/mcp/internal/analytics/mocks"
	"github.com/neo4j/mcp/internal/config"
	"github.com/neo4j/mcp/internal/database"
	"github.com/neo4j/mcp/internal/server"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// startHTTPServer starts a real HTTP MCP server on a random port and returns the server and its base URL.
func startHTTPServer(t *testing.T) (*server.Neo4jMCPServer, string) {
	t.Helper()

	testCFG := dbs.GetDriverConf()
	driver := dbs.GetDriver()

	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to find free port: %v", err)
	}
	port := l.Addr().(*net.TCPAddr).Port
	l.Close()

	cfg := &config.Config{
		URI:                testCFG.URI,
		TransportMode:      config.TransportModeHTTP,
		HTTPHost:           "127.0.0.1",
		HTTPPort:           strconv.Itoa(port),
		HTTPAllowedOrigins: "*",
	}

	validateErr := cfg.Validate()
	if validateErr != nil {
		t.Fatalf("invalid config: %v", validateErr)
	}

	ctrl := gomock.NewController(t)

	dbService, err := database.NewNeo4jService(*driver, cfg.Database, config.TransportModeHTTP, "test-version")
	if err != nil {
		t.Fatalf("failed to create database service: %v", err)
	}

	analyticsService := analytics.NewMockService(ctrl)
	analyticsService.EXPECT().EmitEvent(gomock.Any()).AnyTimes()
	analyticsService.EXPECT().NewStartupEvent(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	analyticsService.EXPECT().IsEnabled().AnyTimes().Return(true)
	analyticsService.EXPECT().NewConnectionInitializedEvent(gomock.Any()).AnyTimes()

	s := server.NewNeo4jMCPServer("test-version", cfg, dbService, analyticsService)
	if s == nil {
		t.Fatal("NewNeo4jMCPServer() returned nil")
	}

	errChan := make(chan error, 1)
	go func() {
		if err := s.Start(); err != nil {
			errChan <- err
		}
	}()

	// Wait for the server to signal readiness, the start goroutine to fail, or a
	// timeout. HTTPServerReady is closed just before ListenAndServe() is called, so
	// give the OS a moment to actually bind after the select unblocks.
	select {
	case <-s.HTTPServerReady:
		time.Sleep(100 * time.Millisecond)
	case startErr := <-errChan:
		t.Fatalf("server failed to start: %v", startErr)
	case <-time.After(10 * time.Second):
		t.Fatal("timed out waiting for HTTP server to be ready")
	}

	baseURL := fmt.Sprintf("http://127.0.0.1:%d", port)

	t.Cleanup(func() {
		stopCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := s.Stop(stopCtx); err != nil {
			t.Errorf("Stop() returned unexpected error: %v", err)
		}
		select {
		case startErr := <-errChan:
			t.Errorf("Start() returned unexpected error: %v", startErr)
		default:
		}
	})

	return s, baseURL
}

func TestHTTPMethodRestrictions(t *testing.T) {
	t.Parallel()

	_, baseURL := startHTTPServer(t)
	testCFG := dbs.GetDriverConf()

	const dbPath = "/db/neo4j/mcp"
	const pingBody = `{"jsonrpc":"2.0","method":"ping","id":1}`
	const methodNotAllowedMsg = "Method Not Allowed: only POST and OPTIONS is supported on /db/{databaseName}/mcp"
	const allowHdr = "POST, OPTIONS"

	tests := []struct {
		name         string
		method       string
		setupReq     func(*http.Request)
		wantStatus   int
		wantBody     string // empty = skip body assertion
		wantAllowHdr string // empty = skip Allow header assertion
	}{
		{
			name:   "POST with valid credentials is accepted",
			method: http.MethodPost,
			setupReq: func(req *http.Request) {
				req.SetBasicAuth(testCFG.Username, testCFG.Password)
				req.Header.Set("Content-Type", "application/json")
			},
			wantStatus: http.StatusOK,
		},
		{
			// CORS middleware intercepts OPTIONS before auth runs (AllowedOrigins: "*"
			// is set on the test server). Preflight returns 204 No Content per spec.
			name:   "OPTIONS returns 204 CORS preflight",
			method: http.MethodOptions,
			setupReq: func(req *http.Request) {
				req.Header.Set("Origin", "http://example.com")
			},
			wantStatus: http.StatusNoContent,
		},
		{
			name:         "GET is rejected",
			method:       http.MethodGet,
			wantStatus:   http.StatusMethodNotAllowed,
			wantBody:     methodNotAllowedMsg,
			wantAllowHdr: allowHdr,
		},
		{
			name:         "PUT is rejected",
			method:       http.MethodPut,
			wantStatus:   http.StatusMethodNotAllowed,
			wantBody:     methodNotAllowedMsg,
			wantAllowHdr: allowHdr,
		},
		{
			name:         "PATCH is rejected",
			method:       http.MethodPatch,
			wantStatus:   http.StatusMethodNotAllowed,
			wantBody:     methodNotAllowedMsg,
			wantAllowHdr: allowHdr,
		},
		{
			name:         "DELETE is rejected",
			method:       http.MethodDelete,
			wantStatus:   http.StatusMethodNotAllowed,
			wantBody:     methodNotAllowedMsg,
			wantAllowHdr: allowHdr,
		},
		{
			name:         "HEAD is rejected",
			method:       http.MethodHead,
			wantStatus:   http.StatusMethodNotAllowed,
			wantAllowHdr: allowHdr,
			// HEAD responses have no body by spec; we check the Allow header only.
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var bodyReader io.Reader
			if tc.method == http.MethodPost {
				bodyReader = strings.NewReader(pingBody)
			}

			req, err := http.NewRequestWithContext(context.Background(), tc.method, baseURL+dbPath, bodyReader)
			require.NoError(t, err)

			if tc.setupReq != nil {
				tc.setupReq(req)
			}

			resp, err := http.DefaultClient.Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, tc.wantStatus, resp.StatusCode)

			if tc.wantBody != "" {
				body, err := io.ReadAll(resp.Body)
				require.NoError(t, err)
				assert.Equal(t, tc.wantBody, strings.TrimSpace(string(body)))
			}

			if tc.wantAllowHdr != "" {
				assert.Equal(t, tc.wantAllowHdr, resp.Header.Get("Allow"))
			}
		})
	}
}
