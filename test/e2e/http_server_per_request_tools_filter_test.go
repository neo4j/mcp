// Copyright (c) "Neo4j"
// Neo4j Sweden AB [http://neo4j.com]

//go:build e2e

package e2e

import (
	"context"
	"encoding/base64"
	"testing"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/neo4j/mcp/test/e2e/helpers"

	"github.com/stretchr/testify/require"
)

func TestHTTPPerRequestToolsFilter(t *testing.T) {
	t.Parallel()

	baseURL := startHTTPModeServer(t)

	t.Run("All tools should be returned from HTTP MCP server", func(t *testing.T) {
		mcpURL := baseURL + "/db/neo4j/mcp"

		cfg := dbs.GetDriverConf()

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		httpClient := newHTTPClient(t, mcpURL, map[string]string{
			"Authorization":   "Basic " + base64.StdEncoding.EncodeToString([]byte(cfg.Username+":"+cfg.Password)),
			"X-Neo4j-MCP-URI": cfg.URI,
		})
		defer httpClient.Close()

		require.NoError(t, httpClient.Start(ctx), "http client failed to start")

		_, err := httpClient.Initialize(ctx, helpers.BuildInitializeRequest())
		require.NoError(t, err, "expected initialize to succeed")

		listToolsResponse, err := httpClient.ListTools(ctx, mcp.ListToolsRequest{})
		require.NoError(t, err, "failed to list tools")

		// Verify we have the expected tools
		if len(listToolsResponse.Tools) != 4 {
			t.Fatal("expected all tools to be available, but got none")
		}
	})

	t.Run("Only read-only tools should be returned when X-Neo4j-MCP-Readonly is set to true", func(t *testing.T) {
		mcpURL := baseURL + "/db/neo4j/mcp"

		cfg := dbs.GetDriverConf()

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		httpClient := newHTTPClient(t, mcpURL, map[string]string{
			"Authorization":        "Basic " + base64.StdEncoding.EncodeToString([]byte(cfg.Username+":"+cfg.Password)),
			"X-Neo4j-MCP-URI":      cfg.URI,
			"X-Neo4j-MCP-Readonly": "true",
		})
		defer httpClient.Close()

		require.NoError(t, httpClient.Start(ctx), "http client failed to start")

		_, err := httpClient.Initialize(ctx, helpers.BuildInitializeRequest())
		require.NoError(t, err, "expected initialize to succeed")

		listToolsResponse, err := httpClient.ListTools(ctx, mcp.ListToolsRequest{})
		require.NoError(t, err, "failed to list tools")

		// Verify we have the expected tools
		if len(listToolsResponse.Tools) != 3 {
			t.Fatalf("expected 3 tools to be available, but got %d", len(listToolsResponse.Tools))
		}
	})

	t.Run("All tools should be returned when X-Neo4j-MCP-Readonly is set to false", func(t *testing.T) {
		mcpURL := baseURL + "/db/neo4j/mcp"

		cfg := dbs.GetDriverConf()

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		httpClient := newHTTPClient(t, mcpURL, map[string]string{
			"Authorization":        "Basic " + base64.StdEncoding.EncodeToString([]byte(cfg.Username+":"+cfg.Password)),
			"X-Neo4j-MCP-URI":      cfg.URI,
			"X-Neo4j-MCP-Readonly": "false",
		})
		defer httpClient.Close()

		require.NoError(t, httpClient.Start(ctx), "http client failed to start")

		_, err := httpClient.Initialize(ctx, helpers.BuildInitializeRequest())
		require.NoError(t, err, "expected initialize to succeed")

		listToolsResponse, err := httpClient.ListTools(ctx, mcp.ListToolsRequest{})
		require.NoError(t, err, "failed to list tools")

		if len(listToolsResponse.Tools) != 4 {
			t.Fatalf("expected all tools to be available, but got %d", len(listToolsResponse.Tools))
		}
	})

	t.Run("All tools should be returned when X-Neo4j-MCP-Readonly is set to mixed case false", func(t *testing.T) {
		mcpURL := baseURL + "/db/neo4j/mcp"

		cfg := dbs.GetDriverConf()

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		httpClient := newHTTPClient(t, mcpURL, map[string]string{
			"Authorization":        "Basic " + base64.StdEncoding.EncodeToString([]byte(cfg.Username+":"+cfg.Password)),
			"X-Neo4j-MCP-URI":      cfg.URI,
			"X-Neo4j-MCP-Readonly": "False",
		})
		defer httpClient.Close()

		require.NoError(t, httpClient.Start(ctx), "http client failed to start")

		_, err := httpClient.Initialize(ctx, helpers.BuildInitializeRequest())
		require.NoError(t, err, "expected initialize to succeed")

		listToolsResponse, err := httpClient.ListTools(ctx, mcp.ListToolsRequest{})
		require.NoError(t, err, "failed to list tools")

		if len(listToolsResponse.Tools) != 4 {
			t.Fatalf("expected all tools to be available, but got %d", len(listToolsResponse.Tools))
		}
	})

	t.Run("Request should fail when X-Neo4j-MCP-Readonly contains an invalid value", func(t *testing.T) {
		mcpURL := baseURL + "/db/neo4j/mcp"

		cfg := dbs.GetDriverConf()

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		httpClient := newHTTPClient(t, mcpURL, map[string]string{
			"Authorization":        "Basic " + base64.StdEncoding.EncodeToString([]byte(cfg.Username+":"+cfg.Password)),
			"X-Neo4j-MCP-URI":      cfg.URI,
			"X-Neo4j-MCP-Readonly": "invalid",
		})
		defer httpClient.Close()

		require.NoError(t, httpClient.Start(ctx), "http client failed to start")

		_, err := httpClient.Initialize(ctx, helpers.BuildInitializeRequest())
		require.Error(t, err, "expected initialize to fail with an invalid readonly header value")
	})
}
