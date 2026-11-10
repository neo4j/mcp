// Copyright (c) "Neo4j"
// Neo4j Sweden AB [http://neo4j.com]

//go:build e2e

package e2e

import (
	"context"
	"net/http"
	"testing"
	"time"

	mcpserver "github.com/neo4j/mcp/internal/server"
	"github.com/neo4j/mcp/test/httpmethods"

	"github.com/stretchr/testify/require"
)

// TestHTTPInvalidCredentials verifies that credentials rejected by Neo4j are answered with
// HTTP 401 and a WWW-Authenticate challenge, before the request reaches the MCP layer.
func TestHTTPInvalidCredentials(t *testing.T) {
	t.Parallel()

	baseURL := startHTTPModeServer(t)

	cfg := dbs.GetDriverConf()
	wrongPassword := "definitely-not-the-password"
	wrongUsername := "no-such-user"

	tests := []struct {
		name           string
		username       *string
		password       *string
		authorization  string
		wantStatus     int
		wantAuthHeader string
	}{
		{
			name:           "When the basic auth password is wrong, the request should be rejected with 401",
			username:       &cfg.Username,
			password:       &wrongPassword,
			wantStatus:     http.StatusUnauthorized,
			wantAuthHeader: "Basic",
		},
		{
			name:           "When the basic auth username is wrong, the request should be rejected with 401",
			username:       &wrongUsername,
			password:       &cfg.Password,
			wantStatus:     http.StatusUnauthorized,
			wantAuthHeader: "Basic",
		},
		{
			name:           "When the bearer token is invalid, the request should be rejected with 401",
			authorization:  "Bearer not-a-valid-token",
			wantStatus:     http.StatusUnauthorized,
			wantAuthHeader: "Bearer",
		},
		{
			name:       "When the basic auth credentials are valid, the request should be accepted",
			username:   &cfg.Username,
			password:   &cfg.Password,
			wantStatus: http.StatusOK,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			headers := map[string]string{
				mcpserver.URIHeader: cfg.URI,
				"Content-Type":      "application/json",
				"Accept":            "application/json, text/event-stream",
			}
			if tc.authorization != "" {
				headers["Authorization"] = tc.authorization
			}

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			client := httpmethods.NewRawHttpClient(headers, baseURL, "/db/neo4j/mcp", tc.username, tc.password)

			resp, _, err := client.Ping(ctx)
			require.NoError(t, err)

			require.Equal(t, tc.wantStatus, resp.StatusCode)
			require.Contains(t, resp.Header.Get("WWW-Authenticate"), tc.wantAuthHeader)
		})
	}
}
