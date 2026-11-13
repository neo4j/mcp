// Copyright (c) "Neo4j"
// Neo4j Sweden AB [http://neo4j.com]

package server

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/neo4j/mcp/internal/config"
	"github.com/neo4j/mcp/internal/mcpcontext"
)

func effectiveRequestTimeout(cfg *config.Config) time.Duration {
	if cfg == nil || cfg.RequestTimeout <= 0 {
		return config.DefaultRequestTimeout
	}
	return cfg.RequestTimeout
}

func formatRequestTimeoutError(ctx context.Context) string {
	if timeout := mcpcontext.GetRequestTimeout(ctx); timeout > 0 {
		return fmt.Sprintf("request timed out after %s", timeout)
	}
	return "request timed out"
}

// isRequestDeadlineExceeded reports whether the request budget expired.
func isRequestDeadlineExceeded(ctx context.Context, err error) bool {
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	return ctx.Err() == context.DeadlineExceeded
}
