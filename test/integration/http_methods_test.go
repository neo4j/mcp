// Copyright (c) "Neo4j"
// Neo4j Sweden AB [http://neo4j.com]

//go:build integration

package integration

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	mockAnalytics "github.com/neo4j/mcp/internal/analytics/mocks"
	"github.com/neo4j/mcp/test/integration/helpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestHTTPMethodRestrictions(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	mockAnalytics := mockAnalytics.NewMockService(ctrl)
	mockAnalytics.EXPECT().EmitEvent(gomock.Any()).AnyTimes()
	mockAnalytics.EXPECT().NewStartupEvent(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	mockAnalytics.EXPECT().IsEnabled().AnyTimes().Return(true)
	mockAnalytics.EXPECT().NewConnectionInitializedEvent(gomock.Any()).AnyTimes()

	_, baseURL := helpers.StartHTTPServer(t, mockAnalytics)

	const dbPath = "/db/neo4j/mcp"
	const pingBody = `{"jsonrpc":"2.0","method":"ping","id":1}`
	const methodNotAllowedMsg = "Method Not Allowed: only POST and OPTIONS is supported on /db/{databaseName}/mcp"
	const allowHdr = "POST, OPTIONS"

	tests := []struct {
		name         string
		method       string
		setupReq     func(*http.Request)
		wantStatus   int
		wantBody     string // empty = skip body assertion
		wantAllowHdr string // empty = skip Allow header assertion
	}{
		{
			name:   "POST with valid credentials is accepted",
			method: http.MethodPost,
			setupReq: func(req *http.Request) {
				req.SetBasicAuth("neo4j", "password")
				req.Header.Set("Content-Type", "application/json")
			},
			wantStatus: http.StatusOK,
		},
		{
			// CORS middleware intercepts OPTIONS before auth runs (AllowedOrigins: "*"
			// is set on the test server). Preflight returns 204 No Content per spec.
			name:   "OPTIONS returns 204 CORS preflight",
			method: http.MethodOptions,
			setupReq: func(req *http.Request) {
				req.Header.Set("Origin", "http://example.com")
			},
			wantStatus: http.StatusNoContent,
		},
		{
			name:         "GET is rejected",
			method:       http.MethodGet,
			wantStatus:   http.StatusMethodNotAllowed,
			wantBody:     methodNotAllowedMsg,
			wantAllowHdr: allowHdr,
		},
		{
			name:         "PUT is rejected",
			method:       http.MethodPut,
			wantStatus:   http.StatusMethodNotAllowed,
			wantBody:     methodNotAllowedMsg,
			wantAllowHdr: allowHdr,
		},
		{
			name:         "PATCH is rejected",
			method:       http.MethodPatch,
			wantStatus:   http.StatusMethodNotAllowed,
			wantBody:     methodNotAllowedMsg,
			wantAllowHdr: allowHdr,
		},
		{
			name:         "DELETE is rejected",
			method:       http.MethodDelete,
			wantStatus:   http.StatusMethodNotAllowed,
			wantBody:     methodNotAllowedMsg,
			wantAllowHdr: allowHdr,
		},
		{
			name:         "HEAD is rejected",
			method:       http.MethodHead,
			wantStatus:   http.StatusMethodNotAllowed,
			wantAllowHdr: allowHdr,
			// HEAD responses have no body by spec; we check the Allow header only.
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var bodyReader io.Reader
			if tc.method == http.MethodPost {
				bodyReader = strings.NewReader(pingBody)
			}

			req, err := http.NewRequestWithContext(context.Background(), tc.method, baseURL+dbPath, bodyReader)
			require.NoError(t, err)

			if tc.setupReq != nil {
				tc.setupReq(req)
			}

			resp, err := http.DefaultClient.Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, tc.wantStatus, resp.StatusCode)

			if tc.wantBody != "" {
				body, err := io.ReadAll(resp.Body)
				require.NoError(t, err)
				assert.Equal(t, tc.wantBody, strings.TrimSpace(string(body)))
			}

			if tc.wantAllowHdr != "" {
				assert.Equal(t, tc.wantAllowHdr, resp.Header.Get("Allow"))
			}
		})
	}
}
