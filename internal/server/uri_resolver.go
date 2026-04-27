// Copyright (c) "Neo4j"
// Neo4j Sweden AB [http://neo4j.com]

package server

import (
	"fmt"
	"net/http"
)

const uriHeader = "X-Neo4j-MCP-URI"

// URIResolver resolves the Neo4j bolt URI for an incoming HTTP request
type URIResolver interface {
	Resolve(r *http.Request) (string, error)
}

// HeaderURIResolver reads the bolt URI from the request header
type HeaderURIResolver struct{}

func (h *HeaderURIResolver) Resolve(r *http.Request) (string, error) {
	uri := r.Header.Get(uriHeader)
	if uri == "" {
		return "", fmt.Errorf("missing required header %s", uriHeader)
	}
	return uri, nil
}
