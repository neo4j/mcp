// Copyright (c) "Neo4j"
// Neo4j Sweden AB [http://neo4j.com]

package database

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/neo4j/mcp/internal/config"
	"github.com/neo4j/mcp/internal/mcpcontext"
	"github.com/neo4j/neo4j-go-driver/v6/neo4j"
)

// applyOptions is a test helper that applies query options to a configuration
// and returns the resulting configuration for inspection.
func applyOptions(options []neo4j.ExecuteQueryConfigurationOption) *neo4j.ExecuteQueryConfiguration {
	config := &neo4j.ExecuteQueryConfiguration{}
	for _, opt := range options {
		opt(config)
	}
	return config
}

// TestBuildQueryOptions covers all combinations of transport mode, auth type,
// and database selection using table-driven tests.
func TestBuildQueryOptions(t *testing.T) {
	tests := []struct {
		name          string
		transportMode config.TransportMode
		setupCtx      func(context.Context) context.Context
		expectedDB    string
		expectAuth    bool
		expectError   string
	}{
		// HTTP mode with bearer token
		{
			name:          "HTTP mode with bearer token and database from context",
			transportMode: config.TransportModeHTTP,
			setupCtx: func(ctx context.Context) context.Context {
				ctx = mcpcontext.WithBearerToken(ctx, "test-bearer-token")
				ctx = mcpcontext.WithDatabaseName(ctx, "explicit-db")
				return ctx
			},
			expectedDB: "explicit-db",
			expectAuth: true,
		},
		{
			name:          "HTTP mode with bearer token but no database in context returns error",
			transportMode: config.TransportModeHTTP,
			setupCtx: func(ctx context.Context) context.Context {
				return mcpcontext.WithBearerToken(ctx, "test-bearer-token")
			},
			expectError: "database name is required in HTTP mode but was not found in context",
		},
		// HTTP mode with basic auth
		{
			name:          "HTTP mode with basic auth and database from context",
			transportMode: config.TransportModeHTTP,
			setupCtx: func(ctx context.Context) context.Context {
				ctx = mcpcontext.WithBasicAuth(ctx, "testuser", "testpass")
				ctx = mcpcontext.WithDatabaseName(ctx, "explicit-db")
				return ctx
			},
			expectedDB: "explicit-db",
			expectAuth: true,
		},
		{
			name:          "HTTP mode with basic auth but no database in context returns error",
			transportMode: config.TransportModeHTTP,
			setupCtx: func(ctx context.Context) context.Context {
				return mcpcontext.WithBasicAuth(ctx, "testuser", "testpass")
			},
			expectError: "database name is required in HTTP mode but was not found in context",
		},
		// HTTP mode without auth
		{
			name:          "HTTP mode without auth and no database in context returns error",
			transportMode: config.TransportModeHTTP,
			setupCtx: func(ctx context.Context) context.Context {
				return ctx
			},
			expectError: "database name is required in HTTP mode but was not found in context",
		},
		{
			name:          "HTTP mode without auth but with database from context",
			transportMode: config.TransportModeHTTP,
			setupCtx: func(ctx context.Context) context.Context {
				return mcpcontext.WithDatabaseName(ctx, "custom-db")
			},
			expectedDB: "custom-db",
			expectAuth: false,
		},
		// STDIO mode (auth and database from context should be ignored)
		{
			name:          "STDIO mode ignores bearer token, uses default database",
			transportMode: config.TransportModeStdio,
			setupCtx: func(ctx context.Context) context.Context {
				ctx = mcpcontext.WithBearerToken(ctx, "test-token")
				ctx = mcpcontext.WithDatabaseName(ctx, "explicit-db")
				return ctx
			},
			expectedDB: "testdb",
			expectAuth: false,
		},
		{
			name:          "STDIO mode ignores basic auth, uses default database",
			transportMode: config.TransportModeStdio,
			setupCtx: func(ctx context.Context) context.Context {
				ctx = mcpcontext.WithBasicAuth(ctx, "user", "pass")
				ctx = mcpcontext.WithDatabaseName(ctx, "explicit-db")
				return ctx
			},
			expectedDB: "testdb",
			expectAuth: false,
		},
		{
			name:          "STDIO mode with no auth, uses default database",
			transportMode: config.TransportModeStdio,
			setupCtx: func(ctx context.Context) context.Context {
				return ctx
			},
			expectedDB: "testdb",
			expectAuth: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &Neo4jService{
				driver:        nil,
				database:      "testdb",
				transportMode: tt.transportMode,
			}

			ctx := tt.setupCtx(context.Background())
			options, err := service.buildQueryOptions(ctx)

			if tt.expectError != "" {
				require.EqualError(t, err, tt.expectError)
				return
			}
			require.NoError(t, err)

			cfg := applyOptions(options)
			assert.Equal(t, tt.expectedDB, cfg.Database)
			assert.Equal(t, tt.expectAuth, cfg.Auth != nil)
		})
	}
}
