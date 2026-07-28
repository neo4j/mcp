// Copyright (c) "Neo4j"
// Neo4j Sweden AB [http://neo4j.com]

package database

import (
	"errors"

	"github.com/neo4j/neo4j-go-driver/v6/neo4j"
)

// Neo4jErrorCode returns the Neo4j error code when err wraps a *neo4j.Neo4jError.
func Neo4jErrorCode(err error) string {
	var neo4jErr *neo4j.Neo4jError
	if errors.As(err, &neo4jErr) && neo4jErr != nil && neo4jErr.Code != "" {
		return neo4jErr.Code
	}
	return ""
}

// ErrorLogAttrs returns slog attributes for err, including neo4j_error_code when present.
func ErrorLogAttrs(err error) []any {
	attrs := []any{"error", err}
	if code := Neo4jErrorCode(err); code != "" {
		attrs = append(attrs, "neo4j_error_code", code)
	}
	return attrs
}
