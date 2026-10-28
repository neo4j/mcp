// Copyright (c) "Neo4j"
// Neo4j Sweden AB [http://neo4j.com]

package server

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"

	"github.com/neo4j/mcp/internal/config"
)

// allowedBoltSchemes is the set of URI schemes accepted in header
var allowedBoltSchemes = map[string]bool{
	"bolt":      true,
	"bolt+s":    true,
	"bolt+ssc":  true,
	"neo4j":     true,
	"neo4j+s":   true,
	"neo4j+ssc": true,
}

var errURINotAllowed = errors.New("URI is not in NEO4J_MCP_HTTP_ALLOWED_URIS")

// URIResolver resolves the Neo4j bolt URI for an incoming HTTP request
type URIResolver interface {
	Resolve(r *http.Request) (string, error)
}

// HeaderURIResolver reads the bolt URI from the request header and checks it against the allowed URIs
type HeaderURIResolver struct {
	allowed config.AllowedURIs
}

func (h *HeaderURIResolver) Resolve(r *http.Request) (string, error) {
	raw := r.Header.Get(URIHeader)
	if raw == "" {
		return "", fmt.Errorf("missing required header %s", URIHeader)
	}

	parsed, err := url.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("invalid URI in header %s: %v", URIHeader, err)
	}

	if !allowedBoltSchemes[parsed.Scheme] {
		return "", fmt.Errorf("invalid URI in header %s: scheme must be one of bolt, bolt+s, bolt+ssc, neo4j, neo4j+s, neo4j+ssc", URIHeader)
	}

	if parsed.Hostname() == "" {
		return "", fmt.Errorf("invalid URI in header %s: host is required", URIHeader)
	}

	if !h.allowed.Allows(parsed.Hostname(), parsed.Port()) {
		return "", errURINotAllowed
	}

	return raw, nil
}
