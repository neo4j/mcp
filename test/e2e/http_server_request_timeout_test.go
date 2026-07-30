// Copyright (c) "Neo4j"
// Neo4j Sweden AB [http://neo4j.com]

//go:build e2e

package e2e

import (
	"context"
	"encoding/base64"
	"testing"
	"time"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/neo4j/mcp/test/e2e/helpers"

	"github.com/stretchr/testify/require"
)

// TestHTTPRequestTimeoutHeaderValidation validate the timeout header validation.
func TestHTTPRequestTimeoutHeaderValidation(t *testing.T) {
	t.Parallel()

	baseURL := startHTTPModeServer(t, "--neo4j-request-timeout", "5s")

	tests := []struct {
		name     string
		timeout  string
		wantBody string
	}{
		{
			name:     "When X-Neo4j-MCP-Request-Timeout exceeds the server maximum, the request should be rejected",
			timeout:  "10s",
			wantBody: "exceeds server maximum",
		},
		{
			name:     "When X-Neo4j-MCP-Request-Timeout is not a valid duration, the request should be rejected",
			timeout:  "not-a-duration",
			wantBody: "must be a valid duration",
		},
		{
			name:     "When X-Neo4j-MCP-Request-Timeout is not positive, the request should be rejected",
			timeout:  "0s",
			wantBody: "must be a positive duration",
		},
		{
			name:     "When X-Neo4j-MCP-Request-Timeout is negative, the request should be rejected",
			timeout:  "-1s",
			wantBody: "must be a positive duration",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			cfg := dbs.GetDriverConf()

			headers := map[string]string{
				"Authorization":               "Basic " + base64.StdEncoding.EncodeToString([]byte(cfg.Username+":"+cfg.Password)),
				"X-Neo4j-MCP-URI":             cfg.URI,
				"X-Neo4j-MCP-Request-Timeout": tc.timeout,
			}

			httpClient := newHTTPClient(t, baseURL+"/db/neo4j/mcp", headers, client.WithSession())
			defer httpClient.Close()

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			err := httpClient.Ping(ctx)
			require.ErrorContains(t, err, "request failed with status 400")
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
				"Authorization":   "Basic " + base64.StdEncoding.EncodeToString([]byte(cfg.Username+":"+cfg.Password)),
				"X-Neo4j-MCP-URI": cfg.URI,
			}
			if tc.timeout != "" {
				headers["X-Neo4j-MCP-Request-Timeout"] = tc.timeout
			}

			httpClient := newHTTPClient(t, baseURL+"/db/neo4j/mcp", headers)
			defer httpClient.Close()

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			require.NoError(t, httpClient.Start(ctx), "http client failed to start")

			_, err := httpClient.Initialize(ctx, helpers.BuildInitializeRequest())
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
				"Authorization":               "Basic " + base64.StdEncoding.EncodeToString([]byte(cfg.Username+":"+cfg.Password)),
				"X-Neo4j-MCP-URI":             cfg.URI,
				"X-Neo4j-MCP-Request-Timeout": tc.timeout,
			}

			httpClient := newHTTPClient(t, baseURL+"/db/neo4j/mcp", headers)
			defer httpClient.Close()

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			require.NoError(t, httpClient.Start(ctx), "http client failed to start")

			_, err := httpClient.Initialize(ctx, helpers.BuildInitializeRequest())
			require.NoError(t, err, "expected initialize to succeed within the %s budget", tc.timeout)

			callToolResponse, err := httpClient.CallTool(ctx, mcp.CallToolRequest{
				Params: mcp.CallToolParams{
					Name: "read-cypher",
					Arguments: map[string]any{
						"query": tc.query,
					},
				},
			})
			require.NoError(t, err)

			if tc.wantError {
				require.True(t, callToolResponse.IsError, "expected a tool error, got: %+v", callToolResponse)

				textContent, ok := callToolResponse.Content[0].(mcp.TextContent)
				require.True(t, ok)
				require.Contains(t, textContent.Text, tc.wantMsg)
				return
			}

			require.False(t, callToolResponse.IsError,
				"expected tool call to succeed, got: %+v", callToolResponse)
		})
	}
}
