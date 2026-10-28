// Copyright (c) "Neo4j"
// Neo4j Sweden AB [http://neo4j.com]

package server

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/neo4j/mcp/internal/config"
	"github.com/neo4j/mcp/internal/mcpcontext"
	"github.com/neo4j/mcp/internal/tools"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

	t.Run("false when only a nested deadline expired", func(t *testing.T) {
		// verifyRequirements probes connectivity under a shorter deadline. When that one
		// expires the request still has budget left, so the underlying error must be
		// reported instead of a request timeout.
		ctx, cancelRequest := context.WithTimeout(context.Background(), time.Minute)
		defer cancelRequest()
		nested, cancelNested := context.WithTimeout(ctx, time.Nanosecond)
		defer cancelNested()
		time.Sleep(time.Millisecond)

		require.Equal(t, context.DeadlineExceeded, nested.Err())
		assert.False(t, isRequestDeadlineExceeded(ctx, nil))
	})
}

func TestRequestTimeoutMiddleware(t *testing.T) {
	const shortTimeout = 20 * time.Millisecond

	newServer := func(timeout time.Duration) *Neo4jMCPServer {
		return &Neo4jMCPServer{config: &config.Config{RequestTimeout: timeout}}
	}

	run := func(s *Neo4jMCPServer, ctx context.Context, next mcp.MethodHandler) (mcp.Result, error) {
		return s.requestTimeoutMiddleware(next)(ctx, "tools/list", nil)
	}

	callToolRequest := func(name string) mcp.Request {
		return &mcp.CallToolRequest{Params: &mcp.CallToolParamsRaw{Name: name}}
	}
	runToolCall := func(s *Neo4jMCPServer, next mcp.MethodHandler) (mcp.Result, error) {
		return s.requestTimeoutMiddleware(next)(context.Background(), "tools/call", callToolRequest("read-cypher"))
	}

	t.Run("applies the configured timeout to the context passed to next", func(t *testing.T) {
		var gotCtx context.Context
		_, err := run(newServer(time.Minute), context.Background(), func(ctx context.Context, _ string, _ mcp.Request) (mcp.Result, error) {
			gotCtx = ctx
			return nil, nil
		})

		require.NoError(t, err)
		assert.Equal(t, time.Minute, mcpcontext.GetRequestTimeout(gotCtx))
		deadline, ok := gotCtx.Deadline()
		require.True(t, ok, "next should receive a context with a deadline")
		assert.WithinDuration(t, time.Now().Add(time.Minute), deadline, 5*time.Second)
	})

	t.Run("falls back to the default when the config has no timeout", func(t *testing.T) {
		var gotCtx context.Context
		_, err := run(newServer(0), context.Background(), func(ctx context.Context, _ string, _ mcp.Request) (mcp.Result, error) {
			gotCtx = ctx
			return nil, nil
		})

		require.NoError(t, err)
		assert.Equal(t, config.DefaultRequestTimeout, mcpcontext.GetRequestTimeout(gotCtx))
	})

	t.Run("keeps the per-request timeout already on the context", func(t *testing.T) {
		ctx := mcpcontext.WithRequestTimeout(context.Background(), 45*time.Second)

		var gotCtx context.Context
		_, err := run(newServer(time.Minute), ctx, func(ctx context.Context, _ string, _ mcp.Request) (mcp.Result, error) {
			gotCtx = ctx
			return nil, nil
		})

		require.NoError(t, err)
		assert.Equal(t, 45*time.Second, mcpcontext.GetRequestTimeout(gotCtx))
		deadline, ok := gotCtx.Deadline()
		require.True(t, ok)
		assert.WithinDuration(t, time.Now().Add(45*time.Second), deadline, 5*time.Second)
	})

	t.Run("returns a timeout error when next exceeds the deadline", func(t *testing.T) {
		res, err := run(newServer(shortTimeout), context.Background(), func(ctx context.Context, _ string, _ mcp.Request) (mcp.Result, error) {
			<-ctx.Done()
			return nil, ctx.Err()
		})

		assert.Nil(t, res)
		require.Error(t, err)
		assert.EqualError(t, err, "request timed out after 20ms")
	})

	t.Run("returns a timeout error when next wraps the deadline error", func(t *testing.T) {
		_, err := run(newServer(shortTimeout), context.Background(), func(ctx context.Context, _ string, _ mcp.Request) (mcp.Result, error) {
			<-ctx.Done()
			return nil, errors.Join(errors.New("query failed"), ctx.Err())
		})

		assert.EqualError(t, err, "request timed out after 20ms")
	})

	t.Run("passes through non-timeout errors unchanged", func(t *testing.T) {
		sentinel := errors.New("boom")

		res, err := run(newServer(time.Minute), context.Background(), func(context.Context, string, mcp.Request) (mcp.Result, error) {
			return nil, sentinel
		})

		assert.Nil(t, res)
		assert.ErrorIs(t, err, sentinel)
	})

	t.Run("passes through the result on success", func(t *testing.T) {
		want := &mcp.ListToolsResult{}

		res, err := run(newServer(time.Minute), context.Background(), func(context.Context, string, mcp.Request) (mcp.Result, error) {
			return want, nil
		})

		require.NoError(t, err)
		assert.Same(t, want, res)
	})

	t.Run("does not turn a successful result into a timeout when the deadline passes during next", func(t *testing.T) {
		want := &mcp.ListToolsResult{}

		res, err := run(newServer(shortTimeout), context.Background(), func(ctx context.Context, _ string, _ mcp.Request) (mcp.Result, error) {
			<-ctx.Done()
			return want, nil
		})

		require.NoError(t, err)
		assert.Same(t, want, res)
	})

	t.Run("tools/call returns a tool error result when next returns the deadline error", func(t *testing.T) {
		res, err := runToolCall(newServer(shortTimeout), func(ctx context.Context, _ string, _ mcp.Request) (mcp.Result, error) {
			<-ctx.Done()
			return nil, ctx.Err()
		})

		require.NoError(t, err)
		toolResult, ok := res.(*mcp.CallToolResult)
		require.True(t, ok)
		assert.True(t, toolResult.IsError)
		require.Len(t, toolResult.Content, 1)
		assert.Equal(t, "request timed out after 20ms", toolResult.Content[0].(*mcp.TextContent).Text)
	})

	t.Run("tools/call returns a tool error result when the deadline passes and next returns an error result", func(t *testing.T) {
		// Tool handlers report failures as error results with a nil error.
		res, err := runToolCall(newServer(shortTimeout), func(ctx context.Context, _ string, _ mcp.Request) (mcp.Result, error) {
			<-ctx.Done()
			return tools.NewToolErrorResult("query failed: context deadline exceeded"), nil
		})

		require.NoError(t, err)
		toolResult, ok := res.(*mcp.CallToolResult)
		require.True(t, ok)
		assert.True(t, toolResult.IsError)
		assert.Equal(t, "request timed out after 20ms", toolResult.Content[0].(*mcp.TextContent).Text)
	})

	t.Run("tools/call passes through the result when the deadline is not reached", func(t *testing.T) {
		want := tools.NewToolErrorResult("some tool failure")

		res, err := runToolCall(newServer(time.Minute), func(context.Context, string, mcp.Request) (mcp.Result, error) {
			return want, nil
		})

		require.NoError(t, err)
		assert.Same(t, want, res)
	})

	t.Run("tools/call passes through non-timeout errors unchanged", func(t *testing.T) {
		sentinel := errors.New("boom")

		_, err := runToolCall(newServer(time.Minute), func(context.Context, string, mcp.Request) (mcp.Result, error) {
			return nil, sentinel
		})

		assert.ErrorIs(t, err, sentinel)
	})
}
