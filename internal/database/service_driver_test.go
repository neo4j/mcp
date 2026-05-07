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

func TestNewNeo4jService_DriverValidation(t *testing.T) {
	driver, err := neo4j.NewDriver("bolt://localhost:7687", neo4j.NoAuth())
	require.NoError(t, err)
	defer func() { _ = driver.Close(context.Background()) }()

	tests := []struct {
		name          string
		driver        neo4j.Driver
		transportMode config.TransportMode
		wantErr       string
	}{
		{
			name:          "nil driver is accepted for HTTP mode",
			driver:        nil,
			transportMode: config.TransportModeHTTP,
		},
		{
			name:          "nil driver is rejected for STDIO mode",
			driver:        nil,
			transportMode: config.TransportModeStdio,
			wantErr:       "driver cannot be nil for STDIO mode",
		},
		{
			name:          "non-nil driver is accepted for HTTP mode",
			driver:        driver,
			transportMode: config.TransportModeHTTP,
		},
		{
			name:          "non-nil driver is accepted for STDIO mode",
			driver:        driver,
			transportMode: config.TransportModeStdio,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, err := NewNeo4jService(tt.driver, "neo4j", tt.transportMode, "test")
			if tt.wantErr != "" {
				require.EqualError(t, err, tt.wantErr)
				assert.Nil(t, svc)
				return
			}
			require.NoError(t, err)
			assert.NotNil(t, svc)
		})
	}
}

func TestGetDriver(t *testing.T) {
	driver, err := neo4j.NewDriver("bolt://localhost:7687", neo4j.NoAuth())
	require.NoError(t, err)
	defer func() { _ = driver.Close(context.Background()) }()

	ctxDriver, err := neo4j.NewDriver("bolt://localhost:7687", neo4j.NoAuth())
	require.NoError(t, err)
	defer func() { _ = ctxDriver.Close(context.Background()) }()

	tests := []struct {
		name       string
		service    *Neo4jService
		ctx        context.Context
		wantDriver neo4j.Driver
		wantErr    string
	}{
		{
			name: "HTTP mode returns driver from context",
			service: &Neo4jService{
				transportMode: config.TransportModeHTTP,
			},
			ctx:        mcpcontext.WithDriver(context.Background(), ctxDriver),
			wantDriver: ctxDriver,
		},
		{
			name: "HTTP mode with no driver in context returns error",
			service: &Neo4jService{
				transportMode: config.TransportModeHTTP,
			},
			ctx:     context.Background(),
			wantErr: "Neo4j driver not available: X-Neo4j-MCP-URI header is required for database operations in HTTP mode",
		},
		{
			name: "STDIO mode returns struct driver",
			service: &Neo4jService{
				driver:        driver,
				transportMode: config.TransportModeStdio,
			},
			ctx:        mcpcontext.WithDriver(context.Background(), ctxDriver),
			wantDriver: driver,
		},
		{
			name: "STDIO mode with no context driver still returns struct driver",
			service: &Neo4jService{
				driver:        driver,
				transportMode: config.TransportModeStdio,
			},
			ctx:        context.Background(),
			wantDriver: driver,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.service.getDriver(tt.ctx)
			if tt.wantErr != "" {
				require.EqualError(t, err, tt.wantErr)
				assert.Nil(t, got)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantDriver, got)
		})
	}
}
