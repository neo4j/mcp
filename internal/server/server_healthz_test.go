// Copyright (c) "Neo4j"
// Neo4j Sweden AB [http://neo4j.com]

package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestHealthzHandler validates the healthzHandler function directly.
func TestHealthzHandler(t *testing.T) {
	tests := []struct {
		name               string
		method             string
		expectedStatusCode int
		expectedBody       string
		expectedAllow      string
		expectedCT         string
	}{
		{
			name:               "GET returns 200 with JSON body",
			method:             http.MethodGet,
			expectedStatusCode: http.StatusOK,
			expectedBody:       `{"status":"ok"}`,
			expectedCT:         "application/json",
		},
		{
			name:               "POST returns 405",
			method:             http.MethodPost,
			expectedStatusCode: http.StatusMethodNotAllowed,
			expectedAllow:      "GET",
		},
		{
			name:               "PUT returns 405",
			method:             http.MethodPut,
			expectedStatusCode: http.StatusMethodNotAllowed,
			expectedAllow:      "GET",
		},
		{
			name:               "DELETE returns 405",
			method:             http.MethodDelete,
			expectedStatusCode: http.StatusMethodNotAllowed,
			expectedAllow:      "GET",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "/healthz", nil)
			rr := httptest.NewRecorder()

			healthzHandler(rr, req)

			res := rr.Result()
			defer res.Body.Close()

			if res.StatusCode != tt.expectedStatusCode {
				t.Errorf("status: expected %d, got %d", tt.expectedStatusCode, res.StatusCode)
			}

			if tt.expectedBody != "" {
				body := rr.Body.String()
				if body != tt.expectedBody {
					t.Errorf("body: expected %q, got %q", tt.expectedBody, body)
				}
			}

			if tt.expectedCT != "" {
				ct := res.Header.Get("Content-Type")
				if ct != tt.expectedCT {
					t.Errorf("Content-Type: expected %q, got %q", tt.expectedCT, ct)
				}
			}

			if tt.expectedAllow != "" {
				allow := res.Header.Get("Allow")
				if allow != tt.expectedAllow {
					t.Errorf("Allow: expected %q, got %q", tt.expectedAllow, allow)
				}
			}
		})
	}
}
