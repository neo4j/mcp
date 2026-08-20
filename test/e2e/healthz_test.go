// Copyright (c) "Neo4j"
// Neo4j Sweden AB [http://neo4j.com]

//go:build e2e

package e2e

import (
	"context"
	"io"
	"net/http"
	"testing"

	"github.com/mark3labs/mcp-go/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestHealthzE2E verifies the /healthz and /mcp ping behaviour of a live HTTP-mode server.
//
// HTTP-mode helpers (server startup, port discovery, env stripping, healthz polling)
// live in http_server_helpers_test.go so other e2e tests can reuse them without
// importing from this file.
func TestHealthzE2E(t *testing.T) {
	t.Parallel()

	baseURL := startHTTPModeServer(t)

	t.Run("GET /healthz returns 200 without credentials", func(t *testing.T) {
		resp, err := http.Get(baseURL + "/healthz") // #nosec G107
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		assert.Equal(t, `{"status":"ok"}`, string(body))
		assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))
	})

	t.Run("POST /mcp ping without credentials returns 401", func(t *testing.T) {
		mcpClient, err := client.NewStreamableHttpClient(baseURL + "/mcp")
		require.NoError(t, err, "failed to create streamable HTTP client")
		defer mcpClient.Close()

		require.NoError(t, mcpClient.Start(context.Background()))

		// Ping sends a JSON-RPC ping to /mcp. pathValidationMiddleware rejects it
		// (path does not match /db/{name}/mcp) before any MCP protocol handling occurs.
		err = mcpClient.Ping(context.Background())
		assert.Error(t, err, "expected rejection when no credentials are provided")
	})
}
