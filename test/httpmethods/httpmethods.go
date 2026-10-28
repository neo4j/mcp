// Copyright (c) "Neo4j"
// Neo4j Sweden AB [http://neo4j.com]

//go:build integration || e2e

package httpmethods

import (
	"context"
	"io"
	"net/http"
	"strings"
)

type RawMCPHTTPClient struct {
	headers  map[string]string
	url      string
	path     string
	username *string
	password *string
}


// NewRawHttpClient creates a RawMCPHTTPClient that issues raw JSON-RPC requests against an MCP
// HTTP server, without going through an MCP client SDK.
func NewRawHttpClient(headers map[string]string, url string, path string, username *string, password *string) *RawMCPHTTPClient {
	RawMCPHTTPClient := &RawMCPHTTPClient{
		headers:  headers,
		url:      url,
		path:     path,
		username: username,
		password: password,
	}
	return RawMCPHTTPClient
}

// Initialize sends a raw JSON-RPC "initialize" request via HTTP POST and returns the HTTP
// response, its trimmed body, and any error.
func (c *RawMCPHTTPClient) Initialize(ctx context.Context) (*http.Response, string, error) {
	return c.SendHttpRequest(ctx, http.MethodPost, `{"jsonrpc":"2.0","method":"initialize","id":1}`)
}

// Ping sends a raw JSON-RPC "ping" request using the client's configured HTTP method (c.method)
// and returns the HTTP response, its trimmed body, and any error.
func (c *RawMCPHTTPClient) Ping(ctx context.Context) (*http.Response, string, error) {
	return c.SendHttpRequest(ctx, http.MethodPost, `{"jsonrpc":"2.0","method":"ping","id":1}`)
}

// SendHttpRequest sends a raw JSON-RPC request over HTTP using the given method and body, and
// returns the HTTP response, its trimmed body, and any error.
func (c *RawMCPHTTPClient) SendHttpRequest(ctx context.Context, method string, jsonBody string) (*http.Response, string, error) {
	bodyReader := strings.NewReader(jsonBody)
	req, err := http.NewRequestWithContext(ctx, method, c.url+c.path, bodyReader)
	if err != nil {
		return nil, "", err
	}

	if c.headers != nil {
		for name, value := range c.headers {
			req.Header.Set(name, value)
		}
	}

	if c.username != nil && c.password != nil {
		req.SetBasicAuth(*c.username, *c.password)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	respBody := strings.TrimSpace(string(body))

	return resp, respBody, err
}
