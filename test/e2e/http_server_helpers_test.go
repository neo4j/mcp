// Copyright (c) "Neo4j"
// Neo4j Sweden AB [http://neo4j.com]

//go:build e2e

// Shared helpers for spinning up the MCP server in HTTP transport mode against a
// random local port. Kept in their own file so any e2e test that needs an HTTP
// server can use them without depending on (or importing from) another test
// file.
package e2e

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/client/transport"

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

	// In HTTP mode the config validation rejects NEO4J_URI, NEO4J_USERNAME, NEO4J_PASSWORD, and
	// NEO4J_DATABASE — the URI, credentials, and database are supplied per-request via the
	// X-Neo4j-MCP-URI header, Auth headers, and URL path respectively.
	// Strip those keys so any locally-set env values don't cause a startup validation error.
	cmd := exec.Command(server, // #nosec G204 -- server is a binary path built by the test harness, not user input
		"--neo4j-transport-mode", "http",
		"--neo4j-http-host", "127.0.0.1",
		"--neo4j-http-port", fmt.Sprintf("%d", port),
		"--neo4j-telemetry", "false",
	)
	cmd.Env = stripEnv(os.Environ(), "NEO4J_URI", "NEO4J_USERNAME", "NEO4J_PASSWORD", "NEO4J_DATABASE")

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

// newHTTPClient builds an MCP streamable-HTTP client forwarding the provided headers.
func newHTTPClient(t *testing.T, mcpURL string, headers map[string]string) *client.Client {
	t.Helper()

	httpTransport, err := transport.NewStreamableHTTP(mcpURL,
		transport.WithHTTPTimeout(15*time.Second),
		transport.WithHTTPHeaders(headers),
	)
	require.NoError(t, err, "failed to build streamable HTTP transport")

	return client.NewClient(httpTransport)
}
