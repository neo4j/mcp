// Copyright (c) "Neo4j"
// Neo4j Sweden AB [http://neo4j.com]

package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHeaderURIResolver_Resolve(t *testing.T) {
	resolver := &HeaderURIResolver{}

	invalidURIErrMsg := "invalid URI in header X-Neo4j-MCP-URI: scheme must be one of bolt, bolt+s, bolt+ssc, neo4j, neo4j+s, neo4j+ssc"

	tests := []struct {
		name      string
		header    string
		setHeader bool
		wantURI   string
		wantErr   string
	}{
		{
			name:      "bolt scheme",
			setHeader: true,
			header:    "bolt://localhost:7687",
			wantURI:   "bolt://localhost:7687",
		},
		{
			name:      "bolt+s scheme",
			setHeader: true,
			header:    "bolt+s://localhost:7687",
			wantURI:   "bolt+s://localhost:7687",
		},
		{
			name:      "bolt+ssc scheme",
			setHeader: true,
			header:    "bolt+ssc://localhost:7687",
			wantURI:   "bolt+ssc://localhost:7687",
		},
		{
			name:      "neo4j scheme",
			setHeader: true,
			header:    "neo4j://localhost:7687",
			wantURI:   "neo4j://localhost:7687",
		},
		{
			name:      "neo4j+s scheme",
			setHeader: true,
			header:    "neo4j+s://localhost:7687",
			wantURI:   "neo4j+s://localhost:7687",
		},
		{
			name:      "neo4j+ssc scheme",
			setHeader: true,
			header:    "neo4j+ssc://localhost:7687",
			wantURI:   "neo4j+ssc://localhost:7687",
		},
		{
			name:      "uri with credentials is passed through",
			setHeader: true,
			header:    "neo4j+s://user:pass@aura.example.com:7687",
			wantURI:   "neo4j+s://user:pass@aura.example.com:7687",
		},
		{
			name:    "absent header returns error",
			wantErr: "missing required header X-Neo4j-MCP-URI",
		},
		{
			name:      "empty header value returns error",
			setHeader: true,
			header:    "",
			wantErr:   "missing required header X-Neo4j-MCP-URI",
		},
		{
			name:      "http scheme rejected",
			setHeader: true,
			header:    "http://localhost:7687",
			wantErr:   invalidURIErrMsg,
		},
		{
			name:      "no scheme rejected",
			setHeader: true,
			header:    "localhost:7687",
			wantErr:   invalidURIErrMsg,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/db/neo4j/mcp", nil)
			if tt.setHeader {
				req.Header.Set(uriHeader, tt.header)
			}

			got, err := resolver.Resolve(req)

			if tt.wantErr != "" {
				require.EqualError(t, err, tt.wantErr)
				assert.Empty(t, got)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantURI, got)
			}
		})
	}
}
