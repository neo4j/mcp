// Copyright (c) "Neo4j"
// Neo4j Sweden AB [http://neo4j.com]

//go:build integration

package integration

import (
	"context"
	"testing"
	"time"

	"github.com/neo4j/mcp/internal/config"
	"github.com/neo4j/mcp/internal/database"
	"github.com/neo4j/mcp/internal/server"
	"github.com/neo4j/mcp/test/integration/helpers"
	"github.com/neo4j/mcp/test/testdb"
	"github.com/neo4j/neo4j-go-driver/v6/neo4j"
	"github.com/stretchr/testify/require"
)

func TestServerLifecycle(t *testing.T) {
	t.Parallel()
	dbs := testdb.GetInstance()
	testCFG := dbs.GetDriverConf()
	testCases := []struct {
		name        string
		config      *config.Config
		expectError bool
	}{
		{
			name: "Neo4jMCPServer should correctly start",
			config: &config.Config{
				URI:           testCFG.URI,
				Username:      testCFG.Username,
				Password:      testCFG.Password,
				Database:      testCFG.Database,
				TransportMode: config.TransportModeStdio,
			},
			expectError: false,
		},
		{
			name: "Neo4jMCPServer should fail to start: invalid host",
			config: &config.Config{
				URI:           "bolt://not-a-valid-host:7687",
				Username:      testCFG.Username,
				Password:      testCFG.Password,
				Database:      testCFG.Database,
				TransportMode: config.TransportModeStdio,
			},
			expectError: true,
		},
		{
			name: "Neo4jMCPServer should fail to start: invalid database name",
			config: &config.Config{
				URI:           testCFG.URI,
				Username:      testCFG.Username,
				Password:      testCFG.Password,
				Database:      "not-a-valid-db-name",
				TransportMode: config.TransportModeStdio,
			},
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			driver, err := neo4j.NewDriver(tc.config.URI, neo4j.BasicAuth(tc.config.Username, tc.config.Password, ""))
			require.NoError(t, err, "failed to create Neo4j driver")

			testContext := helpers.NewTestContext(t, &driver)

			ctx := context.Background()
			defer func() {
				require.NoError(t, driver.Close(ctx), "error closing driver")
			}()

			dbService, err := database.NewNeo4jService(driver, tc.config.Database, tc.config.TransportMode, "test-version")
			require.NoError(t, err, "failed to create database service")

			s := server.NewNeo4jMCPServer("test-version", tc.config, dbService, testContext.AnalyticsService)
			require.NotNil(t, s, "NewNeo4jMCPServer() returned nil")

			errChan := make(chan error, 1)
			go func() {
				errChan <- s.Start()
			}()

			select {
			case startErr := <-errChan:
				if tc.expectError {
					require.Error(t, startErr)
				} else {
					require.NoError(t, startErr, "Start returned an unexpected error")
				}
			case <-time.After(7 * time.Second): // 7 because we have a hardcoded(!) 5 second timeout in the server's Start() for the initial connection, we want to wait a bit longer than that to be sure
				if tc.expectError {
					t.Fatal("expected Start() to return an error but it did not within 7s")
				}
			}
		})
	}

	t.Run("server stop should return no errors", func(t *testing.T) {
		driver, err := neo4j.NewDriverWithContext(testCFG.URI, neo4j.BasicAuth(testCFG.Username, testCFG.Password, ""))
		require.NoError(t, err, "failed to create Neo4j driver")

		testContext := helpers.NewTestContext(t, &driver)
		ctx := context.Background()
		defer func() {
			require.NoError(t, driver.Close(ctx), "error closing driver")
		}()

		dbService, err := database.NewNeo4jService(driver, testCFG.Database, testCFG.TransportMode, "test-version")
		require.NoError(t, err, "failed to create database service")

		testCFGWithTransport := &config.Config{
			URI:           testCFG.URI,
			Username:      testCFG.Username,
			Password:      testCFG.Password,
			Database:      testCFG.Database,
			TransportMode: config.TransportModeStdio,
		}
		s := server.NewNeo4jMCPServer("test-version", testCFGWithTransport, dbService, testContext.AnalyticsService)
		require.NotNil(t, s, "NewNeo4jMCPServer() returned nil")

		errChan := make(chan error, 1)
		go func() {
			errChan <- s.Start()
		}()

		// verify the server hasn't returned an error before we stop it
		select {
		case startErr := <-errChan:
			if startErr != nil {
				t.Fatalf("Start() returned unexpectedly before Stop() was called: %v", startErr)
			}
		case <-time.After(200 * time.Millisecond):
			// server still running, this is expected in a real stdin environment
		}

		stopCtx, cancel := context.WithTimeout(context.Background(), 7*time.Second)
		defer cancel()
		require.NoError(t, s.Stop(stopCtx))
	})
}
