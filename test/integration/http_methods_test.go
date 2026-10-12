// Copyright (c) "Neo4j"
// Neo4j Sweden AB [http://neo4j.com]

//go:build integration

package integration

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"testing"
	"time"

	analytics "github.com/neo4j/mcp/internal/analytics/mocks"
	"github.com/neo4j/mcp/internal/config"
	"github.com/neo4j/mcp/internal/database"
	"github.com/neo4j/mcp/internal/server"
	"github.com/neo4j/mcp/test/httpmethods"
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
		Database:           testCFG.Database,
		TransportMode:      config.TransportModeHTTP,
		HTTPHost:           "127.0.0.1",
		HTTPPort:           strconv.Itoa(port),
		HTTPAllowedOrigins: "*",
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

	noErr := func(t *testing.T, err error) { require.NoError(t, err) }

	dbPath := "/db/neo4j/mcp"
	const methodNotAllowedMsg = "Method Not Allowed: only POST and OPTIONS is supported on /db/{databaseName}/mcp"
	const pingBody = `{"jsonrpc":"2.0","method":"ping","id":1}`

	tests := []struct {
		name         string
		method       string
		path         string
		body         string
		headers      map[string]string
		username     *string
		password     *string
		wantStatus   int
		wantBody     string
		wantAllowHdr string
		assertErr    func(t *testing.T, err error)
	}{
		{
			name:   "POST /db/{db}/mcp with valid credentials returns 200",
			method: http.MethodPost,
			path:   dbPath,
			body:   pingBody,
			headers: map[string]string{
				"Content-Type":   "application/json",
				"Accept":         "application/json, text/event-stream",
				server.URIHeader: testCFG.URI,
			},
			username:   &testCFG.Username,
			password:   &testCFG.Password,
			wantStatus: http.StatusOK,
			assertErr:  noErr,
		},
		{
			// CORS middleware intercepts OPTIONS before auth runs (AllowedOrigins: "*"
			// is set on the test server). Preflight returns 204 No Content per spec.
			name:   "OPTIONS /db/{db}/mcp returns 204 CORS preflight",
			method: http.MethodOptions,
			path:   dbPath,
			headers: map[string]string{
				"Origin": "http://example.com",
			},
			wantStatus: http.StatusNoContent,
			assertErr:  noErr,
		},
		{
			name:         "GET /db/{db}/mcp is rejected",
			method:       http.MethodGet,
			path:         dbPath,
			wantStatus:   http.StatusMethodNotAllowed,
			wantBody:     methodNotAllowedMsg,
			wantAllowHdr: "POST, OPTIONS",
			assertErr:    noErr,
		},
		{
			name:         "PATCH /db/{db}/mcp is rejected",
			method:       http.MethodPatch,
			path:         dbPath,
			wantStatus:   http.StatusMethodNotAllowed,
			wantBody:     methodNotAllowedMsg,
			wantAllowHdr: "POST, OPTIONS",
			assertErr:    noErr,
		},
		{
			name:         "GET /db/{db}/mcp/ (trailing slash) is rejected",
			method:       http.MethodGet,
			path:         dbPath + "/",
			wantStatus:   http.StatusMethodNotAllowed,
			wantBody:     methodNotAllowedMsg,
			wantAllowHdr: "POST, OPTIONS",
			assertErr:    noErr,
		},
		{
			name:         "PATCH /db/{db}/mcp/ (trailing slash) is rejected",
			method:       http.MethodPatch,
			path:         dbPath + "/",
			wantStatus:   http.StatusMethodNotAllowed,
			wantBody:     methodNotAllowedMsg,
			wantAllowHdr: "POST, OPTIONS",
			assertErr:    noErr,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			client := httpmethods.NewRawHttpClient(tc.headers, baseURL, tc.path, tc.username, tc.password)

			resp, respBody, err := client.SendHttpRequest(context.Background(), tc.method, pingBody)

			require.NoError(t, err)

			assert.Equal(t, tc.wantStatus, resp.StatusCode)

			if tc.wantBody != "" {
				assert.Equal(t, tc.wantBody, respBody)
			}

			if tc.wantAllowHdr != "" {
				assert.Equal(t, tc.wantAllowHdr, resp.Header.Get("Allow"))
			}
		})
	}
}

func TestHTTPMode_URIHeader(t *testing.T) {
	t.Parallel()

	_, baseURL := startHTTPServer(t)
	testCFG := dbs.GetDriverConf()
	path := "/db/neo4j/mcp"

	tests := []struct {
		name       string
		headers    map[string]string
		username   *string
		password   *string
		wantStatus int
	}{
		{
			name: "valid X-Neo4j-MCP-URI returns 200",
			headers: map[string]string{
				"Content-Type":   "application/json",
				"Accept":         "application/json, text/event-stream",
				server.URIHeader: testCFG.URI,
			},
			username:   &testCFG.Username,
			password:   &testCFG.Password,
			wantStatus: http.StatusOK,
		},
		{
			name: "missing X-Neo4j-MCP-URI returns 400",
			headers: map[string]string{
				"Content-Type": "application/json",
			},
			username:   &testCFG.Username,
			password:   &testCFG.Password,
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "invalid URI scheme in X-Neo4j-MCP-URI returns 400",
			headers: map[string]string{
				"Content-Type":   "application/json",
				server.URIHeader: "http://localhost:7687",
			},
			username:   &testCFG.Username,
			password:   &testCFG.Password,
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			client := httpmethods.NewRawHttpClient(tc.headers, baseURL, path, tc.username, tc.password)

			resp, respBody, err := client.Ping(context.Background())
			require.NoError(t, err)

			t.Logf("response status=%d body=%s", resp.StatusCode, respBody)

			assert.Equal(t, tc.wantStatus, resp.StatusCode)
		})
	}
}
