// Copyright (c) "Neo4j"
// Neo4j Sweden AB [http://neo4j.com]

//go:build e2e

package e2e

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/mark3labs/mcp-go/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// startHTTPModeServer launches the server binary in HTTP mode on a random free port.
// It polls /healthz until the server is ready and returns the base URL.
// The server process is automatically terminated when the test ends.
func startHTTPModeServer(t *testing.T) string {
	t.Helper()

	port, err := freePort()
	require.NoError(t, err, "could not find a free port")

	baseURL := fmt.Sprintf("http://127.0.0.1:%d", port)

	// In HTTP mode the config validation rejects NEO4J_USERNAME / NEO4J_PASSWORD —
	// credentials are supplied per-request via Basic Auth headers instead.
	// Strip those keys so the e2e suite's env values don't cause a startup error.
	cmd := exec.Command(server,
		"--neo4j-uri", dbs.GetDriverConf().URI,
		"--neo4j-transport-mode", "http",
		"--neo4j-http-host", "127.0.0.1",
		"--neo4j-http-port", fmt.Sprintf("%d", port),
		"--neo4j-telemetry", "false",
	)
	cmd.Env = stripEnv(os.Environ(), "NEO4J_USERNAME", "NEO4J_PASSWORD")

	require.NoError(t, cmd.Start(), "failed to start HTTP server")

	t.Cleanup(func() {
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
			_ = cmd.Wait()
		}
	})

	waitForHealthz(t, baseURL+"/healthz")
	return baseURL
}

// freePort returns an available TCP port on localhost.
func freePort() (int, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer ln.Close()
	return ln.Addr().(*net.TCPAddr).Port, nil
}

// stripEnv returns a copy of env with entries matching any of the given keys removed.
func stripEnv(env []string, keys ...string) []string {
	filtered := make([]string, 0, len(env))
	for _, e := range env {
		skip := false
		for _, key := range keys {
			if strings.HasPrefix(e, key+"=") {
				skip = true
				break
			}
		}
		if !skip {
			filtered = append(filtered, e)
		}
	}
	return filtered
}

// waitForHealthz polls /healthz until it returns HTTP 200 or the deadline expires.
func waitForHealthz(t *testing.T, url string) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := http.Get(url) // #nosec G107
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatalf("server at %s did not become ready within 10s", url)
}

// TestHealthzE2E verifies the /healthz and /mcp ping behaviour of a live HTTP-mode server.
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

		// Ping sends a JSON-RPC ping to /mcp. The auth middleware rejects it
		// with 401 before any MCP protocol handling occurs.
		err = mcpClient.Ping(context.Background())
		assert.Error(t, err, "expected auth rejection when no credentials are provided")
	})
}
