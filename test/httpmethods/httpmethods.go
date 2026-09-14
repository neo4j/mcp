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

func (c *RawMCPHTTPClient) Initialize(ctx context.Context) (*http.Response, string, error) {
	bodyReader := strings.NewReader(`{"jsonrpc":"2.0","method":"initialize","id":1}`)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.url+c.path, bodyReader)

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

func (c *RawMCPHTTPClient) Ping(ctx context.Context) (*http.Response, string, error) {
	bodyReader := strings.NewReader(`{"jsonrpc":"2.0","method":"ping","id":1}`)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.url+c.path, bodyReader)

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
