// Copyright (c) "Neo4j"
// Neo4j Sweden AB [http://neo4j.com]

package server

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/neo4j/mcp/internal/config"
	"github.com/neo4j/mcp/internal/mcpcontext"
	"github.com/stretchr/testify/assert"
)

func TestEffectiveRequestTimeout(t *testing.T) {
	t.Run("nil config falls back to default", func(t *testing.T) {
		assert.Equal(t, config.DefaultRequestTimeout, effectiveRequestTimeout(nil))
	})

	t.Run("zero timeout falls back to default", func(t *testing.T) {
		assert.Equal(t, config.DefaultRequestTimeout, effectiveRequestTimeout(&config.Config{}))
	})

	t.Run("configured timeout is used", func(t *testing.T) {
		cfg := &config.Config{RequestTimeout: 42 * time.Second}
		assert.Equal(t, 42*time.Second, effectiveRequestTimeout(cfg))
	})
}

func TestFormatRequestTimeoutError(t *testing.T) {
	t.Run("includes stored timeout", func(t *testing.T) {
		ctx := mcpcontext.WithRequestTimeout(context.Background(), 30*time.Second)
		assert.Equal(t, "request timed out after 30s", formatRequestTimeoutError(ctx))
	})

	t.Run("generic message without stored timeout", func(t *testing.T) {
		assert.Equal(t, "request timed out", formatRequestTimeoutError(context.Background()))
	})
}

func TestIsRequestDeadlineExceeded(t *testing.T) {
	t.Run("true for deadline exceeded error", func(t *testing.T) {
		assert.True(t, isRequestDeadlineExceeded(context.Background(), context.DeadlineExceeded))
	})

	t.Run("true for expired context", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Nanosecond)
		defer cancel()
		time.Sleep(time.Millisecond)
		assert.True(t, isRequestDeadlineExceeded(ctx, nil))
	})

	t.Run("false for plain cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		assert.False(t, isRequestDeadlineExceeded(ctx, nil))
	})

	t.Run("false for other errors", func(t *testing.T) {
		assert.False(t, isRequestDeadlineExceeded(context.Background(), errors.New("boom")))
	})
}
