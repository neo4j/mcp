// Copyright (c) "Neo4j"
// Neo4j Sweden AB [http://neo4j.com]

//go:build e2e

package e2e

import (
	"context"
	"encoding/base64"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	mcpserver "github.com/neo4j/mcp/internal/server"

	"github.com/stretchr/testify/require"
)

// TestHTTPRequestTimeoutHeaderValidation validate the timeout header validation.
func TestHTTPRequestTimeoutHeaderValidation(t *testing.T) {
	t.Parallel()

	baseURL := startHTTPModeServer(t, "--neo4j-request-timeout", "5s")

	tests := []struct {
		name     string
		timeout  string
		wantStatus int
		wantBody string
	}{
		{
			name:     "When X-Neo4j-MCP-Request-Timeout exceeds the server maximum, the request should be rejected",
			timeout:  "10s",
			wantStatus: http.StatusBadRequest,
			wantBody: "exceeds server maximum",
		},
		{
			name:     "When X-Neo4j-MCP-Request-Timeout is not a valid duration, the request should be rejected",
			timeout:  "not-a-duration",
			wantStatus: http.StatusBadRequest,
			wantBody: "must be a valid duration",
		},
		{
			name:     "When X-Neo4j-MCP-Request-Timeout is not positive, the request should be rejected",
			timeout:  "0s",
			wantStatus: http.StatusBadRequest,
			wantBody: "must be a positive duration",
		},
		{
			name:     "When X-Neo4j-MCP-Request-Timeout is negative, the request should be rejected",
			timeout:  "-1s",
			wantStatus: http.StatusBadRequest,
			wantBody: "must be a positive duration",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			cfg := dbs.GetDriverConf()

			headers := map[string]string{
				"Authorization":         "Basic " + base64.StdEncoding.EncodeToString([]byte(cfg.Username+":"+cfg.Password)),
				mcpserver.URIHeader:     cfg.URI,
				mcpserver.TimeoutHeader: tc.timeout,
				"Content-Type":          "application/json",
				"Accept":                "application/json, text/event-stream",
			}

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			

			req, reqErr := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/db/neo4j/mcp", strings.NewReader(`{"jsonrpc":"2.0","method":"ping","id":1}`))
			require.NoError(t, reqErr)
			for name, value := range headers {
				req.Header.Set(name, value)
			}

			resp, doErr := http.DefaultClient.Do(req)
			require.NoError(t, doErr)
			defer resp.Body.Close()

			require.Equal(t, tc.wantStatus, resp.StatusCode)

			respBody, readErr := io.ReadAll(resp.Body)
			require.NoError(t, readErr)

			err := errors.New(strings.TrimSpace(string(respBody)))
			require.ErrorContains(t, err, tc.wantBody)
		})
	}
}

// TestHTTPRequestTimeoutInitialize verifies that an expired request timeout is
// surfaced as a clear error during the non tool call such initialize
func TestHTTPRequestTimeoutInitialize(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		serverArgs []string
		timeout    string
		wantErr    string
	}{
		{
			name:    "When the header timeout expires, initialize should fail with the timeout error",
			timeout: "1ms",
			wantErr: "request timed out after 1ms",
		},
		{
			name:       "When the server maximum timeout expires, initialize should fail with the timeout error",
			serverArgs: []string{"--neo4j-request-timeout", "1ms"},
			wantErr:    "request timed out after 1ms",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			baseURL := startHTTPModeServer(t, tc.serverArgs...)
			cfg := dbs.GetDriverConf()

			headers := map[string]string{
				"Authorization":     "Basic " + base64.StdEncoding.EncodeToString([]byte(cfg.Username+":"+cfg.Password)),
				mcpserver.URIHeader: cfg.URI,
			}
			if tc.timeout != "" {
				headers[mcpserver.TimeoutHeader] = tc.timeout
			}

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()


			_, err := newHTTPClient(t, ctx, baseURL+"/db/neo4j/mcp", headers)
			require.ErrorContains(t, err, tc.wantErr)
		})
	}
}

// TestHTTPRequestTimeoutToolCall verifies that a tool call exceeding the request
// timeout budget is cancelled and reported as a tool error, while a call within
// budget succeeds.
func TestHTTPRequestTimeoutToolCall(t *testing.T) {
	t.Parallel()

	baseURL := startHTTPModeServer(t)

	tests := []struct {
		name      string
		timeout   string
		query     string
		wantError bool
		wantMsg   string
	}{
		{
			name:      "When the tool call exceeds the request timeout, a timeout tool error should be returned",
			timeout:   "2s",
			query:     "CALL apoc.util.sleep(5000) RETURN 1 AS n",
			wantError: true,
			wantMsg:   "request timed out after 2s",
		},
		{
			name:      "When the tool call completes within the request timeout, it should succeed",
			timeout:   "5s",
			query:     "RETURN 1 AS n",
			wantError: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			cfg := dbs.GetDriverConf()

			headers := map[string]string{
				"Authorization":         "Basic " + base64.StdEncoding.EncodeToString([]byte(cfg.Username+":"+cfg.Password)),
				mcpserver.URIHeader:     cfg.URI,
				mcpserver.TimeoutHeader: tc.timeout,
			}

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			session, err := newHTTPClient(t, ctx, baseURL+"/db/neo4j/mcp", headers)
			require.NoError(t, err, "expected initialize to succeed within the %s budget", tc.timeout)
			defer session.Close()

			callToolResponse, err := session.CallTool(ctx, &mcp.CallToolParams{
				Name: "read-cypher",
				Arguments: map[string]any{
					"query": tc.query,
				},
			})
			require.NoError(t, err)

			if tc.wantError {
				require.True(t, callToolResponse.IsError, "expected a tool error, got: %+v", callToolResponse)

				textContent, ok := callToolResponse.Content[0].(*mcp.TextContent)
				require.True(t, ok)
				require.Contains(t, textContent.Text, tc.wantMsg)
				return
			}

			require.False(t, callToolResponse.IsError,
				"expected tool call to succeed, got: %+v", callToolResponse)
		})
	}
}
