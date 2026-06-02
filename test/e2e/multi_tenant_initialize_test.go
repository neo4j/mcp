// Copyright (c) "Neo4j"
// Neo4j Sweden AB [http://neo4j.com]

//go:build e2e

package e2e

import (
	"context"
	"testing"
	"time"

	"github.com/neo4j/mcp/test/e2e/helpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMultiTenantHTTPInitializeIsolation verifies that, in HTTP transport mode,
// two consecutive initialize requests from different "tenants" do not affect each
// other. In particular, a first initialize that fails because the caller supplied
// wrong credentials must not leave the server in a state that prevents a second,
// well-formed initialize from succeeding.
//
// HTTP mode is multi-tenant by design: each request carries its own bolt URI
// (via the X-Neo4j-MCP-URI header) and Neo4j credentials (via Basic Auth), and
// the server runs verifyRequirements against those per-request credentials on
// every initialize. Tenant A's failure must therefore be fully isolated from
// tenant B's request.
func TestMultiTenantHTTPInitializeIsolation(t *testing.T) {
	t.Parallel()

	baseURL := startHTTPModeServer(t)
	// The Neo4j test instance is reachable at the "neo4j" database. The path is
	// validated by pathValidationMiddleware which only accepts /db/{name}/mcp.
	mcpURL := baseURL + "/db/neo4j/mcp"

	cfg := dbs.GetDriverConf()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Tenant A — valid URI header but wrong password. The auth middleware
	// happily forwards the credentials, neo4jDriverMiddleware builds a per-request
	// driver, and the initialize hook then runs verifyRequirements which fails
	// because Neo4j rejects the password. The initialize call must propagate
	// that failure back to the client.
	wrongClient := newHTTPClient(t, mcpURL, cfg.Username, "definitely-not-the-password", cfg.URI)
	defer wrongClient.Close()

	require.NoError(t, wrongClient.Start(ctx), "wrong-tenant client failed to start")
	_, err := wrongClient.Initialize(ctx, helpers.BuildInitializeRequest())
	require.Error(t, err, "expected initialize with wrong credentials to fail")

	// Tenant B — same server, correct credentials. If the server correctly
	// isolates per-request state, this initialize must succeed even though the
	// previous one (from a different tenant) failed.
	rightClient := newHTTPClient(t, mcpURL, cfg.Username, cfg.Password, cfg.URI)
	defer rightClient.Close()

	require.NoError(t, rightClient.Start(ctx), "right-tenant client failed to start")
	initResp, err := rightClient.Initialize(ctx, helpers.BuildInitializeRequest())
	require.NoError(t, err, "expected initialize with right credentials to succeed after wrong tenant's failure")

	assert.Equal(t, "neo4j-mcp", initResp.ServerInfo.Name)
	assert.NotNil(t, initResp.Capabilities, "server must advertise capabilities after a successful initialize")
	assert.NotNil(t, initResp.Capabilities.Tools, "server must advertise tools after a successful initialize")
}
