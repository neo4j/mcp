// Copyright (c) "Neo4j"
// Neo4j Sweden AB [http://neo4j.com]

package database

import (
	"errors"
	"slices"

	"github.com/neo4j/neo4j-go-driver/v6/neo4j"
)

// ErrConnectionFailed replaces driver connectivity errors in HTTP mode so that network details are not returned to the caller.
var ErrConnectionFailed = errors.New("unable to connect to the Neo4j instance")

// IsConnectivityError reports whether err is a driver connectivity error, including one reported after retries.
func IsConnectivityError(err error) bool {
	var connectivityErr *neo4j.ConnectivityError
	if errors.As(err, &connectivityErr) {
		return true
	}
	// TransactionExecutionLimit does not implement Unwrap, so the retried errors are checked explicitly.
	var limitErr *neo4j.TransactionExecutionLimit
	return errors.As(err, &limitErr) && slices.ContainsFunc(limitErr.Errors, IsConnectivityError)
}

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
