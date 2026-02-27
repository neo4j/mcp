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
	"go.uber.org/mock/gomock"
)

// startHTTPServer starts a real HTTP MCP server on a random port and returns the server,
// its base URL, and a channel that receives any error from Start().
// The caller is responsible for stopping the server via t.Cleanup or directly.
func startHTTPServer(t *testing.T) (*server.Neo4jMCPServer, string, chan error) {
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
		URI:           testCFG.URI,
		Database:      testCFG.Database,
		TransportMode: config.TransportModeHTTP,
		HTTPHost:      "127.0.0.1",
		HTTPPort:      strconv.Itoa(port),
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

	// Wait for the server to signal readiness, then give it a moment to bind.
	// HTTPServerReady closes before ListenAndServe() is called.
	for range s.HTTPServerReady { //nolint:all
	}
	time.Sleep(100 * time.Millisecond)

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

	return s, baseURL, errChan
}

func TestHTTPMethodRestrictions(t *testing.T) {
	t.Parallel()

	_, baseURL, _ := startHTTPServer(t)

	t.Run("GET /mcp returns 405 with Allow header", func(t *testing.T) {
		req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, baseURL+"/mcp", nil)
		if err != nil {
			t.Fatalf("failed to create request: %v", err)
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusMethodNotAllowed {
			t.Errorf("expected 405 Method Not Allowed, got %d", resp.StatusCode)
		}
		if allow := resp.Header.Get("Allow"); allow != "POST, OPTIONS" {
			t.Errorf("expected Allow: POST, OPTIONS, got %q", allow)
		}
	})
}
