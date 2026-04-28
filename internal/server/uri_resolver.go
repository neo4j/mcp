// Copyright (c) "Neo4j"
// Neo4j Sweden AB [http://neo4j.com]

package server

import (
	"fmt"
	"net/http"
	"net/url"
)

const uriHeader = "X-Neo4j-MCP-URI"

// allowedBoltSchemes is the set of URI schemes accepted in header
var allowedBoltSchemes = map[string]bool{
	"bolt":      true,
	"bolt+s":    true,
	"bolt+ssc":  true,
	"neo4j":     true,
	"neo4j+s":   true,
	"neo4j+ssc": true,
}

// URIResolver resolves the Neo4j bolt URI for an incoming HTTP request
type URIResolver interface {
	Resolve(r *http.Request) (string, error)
}

// HeaderURIResolver reads the bolt URI from the request header
type HeaderURIResolver struct{}

func (h *HeaderURIResolver) Resolve(r *http.Request) (string, error) {
	raw := r.Header.Get(uriHeader)
	if raw == "" {
		return "", fmt.Errorf("missing required header %s", uriHeader)
	}

	parsed, err := url.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("invalid URI in header %s: %v", uriHeader, err)
	}

	if !allowedBoltSchemes[parsed.Scheme] {
		return "", fmt.Errorf("invalid URI in header %s: scheme must be one of bolt, bolt+s, bolt+ssc, neo4j, neo4j+s, neo4j+ssc", uriHeader)
	}

	return raw, nil
}
