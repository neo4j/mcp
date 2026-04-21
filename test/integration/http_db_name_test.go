// Copyright (c) "Neo4j"
// Neo4j Sweden AB [http://neo4j.com]

//go:build integration

package integration

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestHTTPDatabaseNameValidation covers the db-name validation layer
func TestHTTPDatabaseNameValidation(t *testing.T) {
	t.Parallel()

	_, baseURL := startHTTPServer(t)

	const pingBody = `{"jsonrpc":"2.0","method":"ping","id":1}`
	const invalidNameMsg = "Bad Request: Invalid database name"
	const notFoundMsg = "Not Found: This server only handles requests to /db/{databaseName}/mcp"

	withAuth := func(req *http.Request) {
		req.SetBasicAuth("neo4j", "password")
		req.Header.Set("Content-Type", "application/json")
	}

	tests := []struct {
		name       string
		path       string
		setupReq   func(*http.Request)
		wantStatus int
		wantBody   string // empty = skip body assertion
	}{
		// Valid name formats — must pass the validation layer
		{
			name:       "3-char minimum name is accepted",
			path:       "/db/abc/mcp",
			setupReq:   withAuth,
			wantStatus: http.StatusOK,
		},
		{
			name:       "hyphenated name is accepted",
			path:       "/db/my-db/mcp",
			setupReq:   withAuth,
			wantStatus: http.StatusOK,
		},
		{
			name:       "dot-separated name is accepted",
			path:       "/db/my.db/mcp",
			setupReq:   withAuth,
			wantStatus: http.StatusOK,
		},
		{
			// 63 characters — maximum allowed length
			name:       "63-char name is accepted",
			path:       "/db/abcdefghij-abcdefghij-abcdefghij-abcdefghij-abcdefghij-abcdefgh/mcp",
			setupReq:   withAuth,
			wantStatus: http.StatusOK,
		},
		// Invalid names — rejected by dbNameMiddleware with 400
		{
			name:       "2-char name is too short",
			path:       "/db/ab/mcp",
			wantStatus: http.StatusBadRequest,
			wantBody:   invalidNameMsg,
		},
		{
			name:       "leading underscore is invalid",
			path:       "/db/_private/mcp",
			wantStatus: http.StatusBadRequest,
			wantBody:   invalidNameMsg,
		},
		{
			name:       "name starting with 'system' is reserved",
			path:       "/db/systemdb/mcp",
			wantStatus: http.StatusBadRequest,
			wantBody:   invalidNameMsg,
		},
		{
			name:       "dollar sign is invalid",
			path:       "/db/my$db/mcp",
			wantStatus: http.StatusBadRequest,
			wantBody:   invalidNameMsg,
		},
		{
			name:       "leading hyphen is invalid",
			path:       "/db/-bad/mcp",
			wantStatus: http.StatusBadRequest,
			wantBody:   invalidNameMsg,
		},
		{
			name:       "trailing hyphen is invalid",
			path:       "/db/bad-/mcp",
			wantStatus: http.StatusBadRequest,
			wantBody:   invalidNameMsg,
		},
		// Malformed paths — rejected by pathValidationMiddleware with 404
		{
			name:       "missing /db prefix",
			path:       "/mcp",
			wantStatus: http.StatusNotFound,
			wantBody:   notFoundMsg,
		},
		{
			name:       "missing /mcp suffix",
			path:       "/db/neo4j",
			wantStatus: http.StatusNotFound,
			wantBody:   notFoundMsg,
		},
		{
			name:       "extra segment after /mcp",
			path:       "/db/neo4j/mcp/extra",
			wantStatus: http.StatusNotFound,
			wantBody:   notFoundMsg,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequestWithContext(
				context.Background(),
				http.MethodPost,
				fmt.Sprintf("%s%s", baseURL, tt.path),
				strings.NewReader(pingBody),
			)
			require.NoError(t, err)

			if tt.setupReq != nil {
				tt.setupReq(req)
			}

			resp, err := http.DefaultClient.Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, tt.wantStatus, resp.StatusCode)

			if tt.wantBody != "" {
				bodyBytes, err := io.ReadAll(resp.Body)
				require.NoError(t, err)
				assert.Equal(t, tt.wantBody, strings.TrimSpace(string(bodyBytes)))
			}
		})
	}
}

// // TestHTTPDatabaseNameInToolExecution validates the full request chain:
// // URL parse → context → buildQueryOptions → Neo4j query.
// // This requires a live database connection.
// func TestHTTPDatabaseNameInToolExecution(t *testing.T) {
// 	t.Parallel()

// 	_, baseURL := startHTTPServer(t)
// 	testCFG := dbs.GetDriverConf()

// 	const toolCallBody = `{"jsonrpc":"2.0","method":"tools/call","params":{"name":"run-cypher","arguments":{"query":"RETURN 1 AS n","write":false}},"id":1}`

// 	req, err := http.NewRequestWithContext(
// 		context.Background(),
// 		http.MethodPost,
// 		baseURL+"/db/neo4j/mcp",
// 		strings.NewReader(toolCallBody),
// 	)
// 	require.NoError(t, err)
// 	req.SetBasicAuth(testCFG.Username, testCFG.Password)
// 	req.Header.Set("Content-Type", "application/json")

// 	resp, err := http.DefaultClient.Do(req)
// 	require.NoError(t, err)
// 	defer resp.Body.Close()

// 	require.Equal(t, http.StatusOK, resp.StatusCode)

// 	bodyBytes, err := io.ReadAll(resp.Body)
// 	require.NoError(t, err)

// 	var rpcResp map[string]any
// 	require.NoError(t, json.Unmarshal(bodyBytes, &rpcResp), "response must be valid JSON: %s", string(bodyBytes))
// 	assert.Contains(t, rpcResp, "result", "expected result field, not error: %s", string(bodyBytes))
// 	assert.NotContains(t, rpcResp, "error", "expected no error field: %s", string(bodyBytes))
// }
