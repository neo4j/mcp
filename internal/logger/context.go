// Copyright (c) "Neo4j"
// Neo4j Sweden AB [http://neo4j.com]

package logger

import (
	"context"

	"github.com/neo4j/mcp/internal/mcpcontext"
)

// AppendRequestInfo returns request-scoped slog attributes from ctx.
// Append to any slog call: slog.Info("msg", append(logger.AppendRequestInfo(ctx), "key", val)...).
func AppendRequestInfo(ctx context.Context) []any {
	var attrs []any

	if id, ok := mcpcontext.GetRequestID(ctx); ok {
		attrs = append(attrs, "request_id", id)
	}
	if db, ok := mcpcontext.GetDatabaseName(ctx); ok {
		attrs = append(attrs, "database", db)
	}
	if target, ok := mcpcontext.GetNeo4jTarget(ctx); ok {
		attrs = append(attrs, "neo4j_target", target)
	}
	if dbID, ok := mcpcontext.GetDBID(ctx); ok && dbID != "" {
		attrs = append(attrs, "db_id", dbID)
	}

	return attrs
}

// AuthType returns the authentication mechanism in use without exposing credentials.
func AuthType(ctx context.Context) string {
	if _, ok := mcpcontext.GetBearerToken(ctx); ok {
		return "bearer"
	}
	if mcpcontext.HasAuth(ctx) {
		return "basic"
	}
	return "none"
}
