// Copyright (c) "Neo4j"
// Neo4j Sweden AB [http://neo4j.com]

package server_test

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	analyticsReal "github.com/neo4j/mcp/internal/analytics"
	analytics "github.com/neo4j/mcp/internal/analytics/mocks"
	"github.com/neo4j/mcp/internal/config"
	db "github.com/neo4j/mcp/internal/database/mocks"
	"github.com/neo4j/mcp/internal/server"
	"github.com/neo4j/neo4j-go-driver/v6/neo4j"
	"go.uber.org/mock/gomock"
)

func TestNewNeo4jMCPServer(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	cfg := &config.Config{
		URI:           "bolt://test-host:7687",
		Username:      "neo4j",
		Password:      "password",
		Database:      "neo4j",
		Tools:         config.AvailableTools,
		TransportMode: config.TransportModeStdio,
	}

	analyticsService := analytics.NewMockService(ctrl)
	analyticsService.EXPECT().IsEnabled().AnyTimes().Return(true)
	analyticsService.EXPECT().EmitEvent(gomock.Any()).AnyTimes()
	analyticsService.EXPECT().NewStartupEvent(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	analyticsService.EXPECT().NewConnectionInitializedEvent(gomock.Any()).AnyTimes()

	t.Run("starts server successfully", func(t *testing.T) {
		withFreshStdin(t)

		mockDB := db.NewMockService(ctrl)

		s := server.NewNeo4jMCPServer("test-version", cfg, mockDB, analyticsService)

		if s == nil {
			t.Fatal("NewNeo4jMCPServer() expected non-nil server, got nil")
		}

		err := s.Start()

		if err != nil {
			t.Errorf("Start() unexpected error = %v", err)
		}
	})
	t.Run("stops server successfully", func(t *testing.T) {
		withFreshStdin(t)

		mockDB := db.NewMockService(ctrl)

		s := server.NewNeo4jMCPServer("test-version", cfg, mockDB, analyticsService)

		if s == nil {
			t.Fatal("NewNeo4jMCPServer() expected non-nil server, got nil")
		}

		err := s.Start()
		if err != nil {
			t.Errorf("Start() unexpected error = %v", err)
		}
	})
}

func TestInitializeRequestHook(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	cfg := &config.Config{
		URI:           "bolt://test-host:7687",
		Username:      "neo4j",
		Password:      "password",
		Database:      "neo4j",
		Tools:         config.AvailableTools,
		TransportMode: config.TransportModeStdio,
	}

	analyticsService := analytics.NewMockService(ctrl)
	analyticsService.EXPECT().IsEnabled().AnyTimes().Return(true)
	analyticsService.EXPECT().EmitEvent(gomock.Any()).AnyTimes()
	analyticsService.EXPECT().NewStartupEvent(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	analyticsService.EXPECT().NewConnectionInitializedEvent(gomock.Any()).AnyTimes()
	t.Run("initialize successful", func(t *testing.T) {
		withFreshStdin(t)

		mockDB := db.NewMockService(ctrl)
		mockDB.EXPECT().ExecuteReadQuery(gomock.Any(), "RETURN 1 as first", gomock.Any()).Times(1).Return([]*neo4j.Record{
			{
				Keys: []string{"first"},
				Values: []any{
					int64(1),
				},
			},
		}, nil)
		checkApocMetaSchemaQuery := "SHOW PROCEDURES YIELD name WHERE name = 'apoc.meta.schema' RETURN count(name) > 0 AS apocMetaSchemaAvailable"
		mockDB.EXPECT().ExecuteReadQuery(gomock.Any(), checkApocMetaSchemaQuery, gomock.Any()).Times(1).Return([]*neo4j.Record{
			{
				Keys: []string{"apocMetaSchemaAvailable"},
				Values: []any{
					bool(true),
				},
			},
		}, nil)
		gdsVersionQuery := "RETURN gds.version() as gdsVersion"
		mockDB.EXPECT().ExecuteReadQuery(gomock.Any(), gdsVersionQuery, gomock.Any()).Times(1).Return([]*neo4j.Record{
			{
				Keys: []string{"gdsVersion"},
				Values: []any{
					string("2.22.0"),
				},
			},
		}, nil)
		mockDB.EXPECT().ExecuteReadQuery(gomock.Any(), "CALL dbms.components()", gomock.Any()).Times(1)
		s := server.NewNeo4jMCPServer("test-version", cfg, mockDB, analyticsService)

		if s == nil {
			t.Fatal("NewNeo4jMCPServer() expected non-nil server, got nil")
		}
		err := s.Start()
		if err != nil {
			t.Errorf("error while starting the MCP Server")
		}
		_, err = connectInProcessClient(t, context.Background(), s.MCPServer)
		if err != nil {
			t.Fatalf("Expect no error during initialization, got: %s", err.Error())
		}
	})
	t.Run("starts server should fails when no connection can be established", func(t *testing.T) {
		withFreshStdin(t)

		mockDB := db.NewMockService(ctrl)
		mockDB.EXPECT().ExecuteReadQuery(gomock.Any(), gomock.Any(), gomock.Any()).Times(2).Return(nil, fmt.Errorf("connection error"))
		s := server.NewNeo4jMCPServer("test-version", cfg, mockDB, analyticsService)

		if s == nil {
			t.Fatal("NewNeo4jMCPServer() expected non-nil server, got nil")
		}
		err := s.Start()
		if err != nil {
			t.Errorf("error while starting the MCP Server")
		}
		_, err = connectInProcessClient(t, context.Background(), s.MCPServer)
		if err == nil {
			t.Fatal("Expect error during initialization, when no connection can be established, got nil")
		}

	})
	t.Run("starts server should fail when test query returns unexpected result", func(t *testing.T) {
		withFreshStdin(t)

		mockDB := db.NewMockService(ctrl)
		mockDB.EXPECT().ExecuteReadQuery(gomock.Any(), "RETURN 1 as first", gomock.Any()).Times(2).Return([]*neo4j.Record{
			{
				Keys:   []string{"first"},
				Values: []any{int64(2)}, // Return a value other than 1
			},
		}, nil)
		s := server.NewNeo4jMCPServer("test-version", cfg, mockDB, analyticsService)

		if s == nil {
			t.Fatal("NewNeo4jMCPServer() expected non-nil server, got nil")
		}

		err := s.Start()
		if err != nil {
			t.Errorf("error while starting the MCP Server")
		}
		_, err = connectInProcessClient(t, context.Background(), s.MCPServer)
		if err == nil {
			t.Fatal("Expect error during initialization, when unexpected results are returned, got nil")
		}
	})

	t.Run("starts server successfully if GDS is not found", func(t *testing.T) {
		withFreshStdin(t)

		mockDB := db.NewMockService(ctrl)
		mockDB.EXPECT().ExecuteReadQuery(gomock.Any(), "RETURN 1 as first", gomock.Any()).Times(1).Return([]*neo4j.Record{
			{
				Keys: []string{"first"},
				Values: []any{
					int64(1),
				},
			},
		}, nil)
		checkApocMetaSchemaQuery := "SHOW PROCEDURES YIELD name WHERE name = 'apoc.meta.schema' RETURN count(name) > 0 AS apocMetaSchemaAvailable"
		mockDB.EXPECT().ExecuteReadQuery(gomock.Any(), checkApocMetaSchemaQuery, gomock.Any()).Times(1).Return([]*neo4j.Record{
			{
				Keys: []string{"apocMetaSchemaAvailable"},
				Values: []any{
					bool(true),
				},
			},
		}, nil)
		gdsVersionQuery := "RETURN gds.version() as gdsVersion"
		mockDB.EXPECT().ExecuteReadQuery(gomock.Any(), gdsVersionQuery, gomock.Any()).Times(1).Return(nil, fmt.Errorf("Unknown function 'gds.version'"))
		mockDB.EXPECT().ExecuteReadQuery(gomock.Any(), "CALL dbms.components()", gomock.Any()).Times(1)

		s := server.NewNeo4jMCPServer("test-version", cfg, mockDB, analyticsService)

		if s == nil {
			t.Fatal("NewNeo4jMCPServer() expected non-nil server, got nil")
		}

		err := s.Start()

		if err != nil {
			t.Errorf("error while starting the MCP Server")
		}
		_, err = connectInProcessClient(t, context.Background(), s.MCPServer)
		if err != nil {
			t.Fatalf("Expect no error during initialization, got: %s", err.Error())
		}
	})

	t.Run("skips GDS verification when list-gds-procedures is not enabled", func(t *testing.T) {
		withFreshStdin(t)

		cfgWithoutGDS := &config.Config{
			URI:           "bolt://test-host:7687",
			Username:      "neo4j",
			Password:      "password",
			Database:      "neo4j",
			Tools:         []string{"read-cypher", "write-cypher", "get-schema"},
			TransportMode: config.TransportModeStdio,
		}
		mockDB := db.NewMockService(ctrl)
		mockDB.EXPECT().ExecuteReadQuery(gomock.Any(), "RETURN 1 as first", gomock.Any()).Times(1).Return([]*neo4j.Record{
			{
				Keys:   []string{"first"},
				Values: []any{int64(1)},
			},
		}, nil)
		checkApocMetaSchemaQuery := "SHOW PROCEDURES YIELD name WHERE name = 'apoc.meta.schema' RETURN count(name) > 0 AS apocMetaSchemaAvailable"
		mockDB.EXPECT().ExecuteReadQuery(gomock.Any(), checkApocMetaSchemaQuery, gomock.Any()).Times(1).Return([]*neo4j.Record{
			{
				Keys:   []string{"apocMetaSchemaAvailable"},
				Values: []any{bool(true)},
			},
		}, nil)
		mockDB.EXPECT().ExecuteReadQuery(gomock.Any(), "CALL dbms.components()", gomock.Any()).Times(1)

		s := server.NewNeo4jMCPServer("test-version", cfgWithoutGDS, mockDB, analyticsService)
		err := s.Start()
		if err != nil {
			t.Errorf("error while starting the MCP Server")
		}
		_, err = connectInProcessClient(t, context.Background(), s.MCPServer)
		if err != nil {
			t.Fatalf("Expect no error during initialization, got: %s", err.Error())
		}
	})
}

func TestNewNeo4jMCPServerEvents(t *testing.T) {
	// move to initialize tests
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	cfg := &config.Config{
		URI:           "bolt://test-host:7687",
		Username:      "neo4j",
		Password:      "password",
		Database:      "neo4j",
		Tools:         config.AvailableTools,
		TransportMode: config.TransportModeStdio,
	}

	mockDB := db.NewMockService(ctrl)
	mockDB.EXPECT().ExecuteReadQuery(gomock.Any(), "RETURN 1 as first", gomock.Any()).AnyTimes().Return([]*neo4j.Record{
		{
			Keys: []string{"first"},
			Values: []any{
				int64(1),
			},
		},
	}, nil)
	checkApocMetaSchemaQuery := "SHOW PROCEDURES YIELD name WHERE name = 'apoc.meta.schema' RETURN count(name) > 0 AS apocMetaSchemaAvailable"
	mockDB.EXPECT().ExecuteReadQuery(gomock.Any(), checkApocMetaSchemaQuery, gomock.Any()).AnyTimes().Return([]*neo4j.Record{
		{
			Keys: []string{"apocMetaSchemaAvailable"},
			Values: []any{
				bool(true),
			},
		},
	}, nil)
	gdsVersionQuery := "RETURN gds.version() as gdsVersion"
	mockDB.EXPECT().ExecuteReadQuery(gomock.Any(), gdsVersionQuery, gomock.Any()).AnyTimes().Return([]*neo4j.Record{
		{
			Keys: []string{"gdsVersion"},
			Values: []any{
				string("2.22.0"),
			},
		},
	}, nil)
	mockDB.EXPECT().ExecuteReadQuery(gomock.Any(), "CALL dbms.components()", gomock.Any()).Times(1).Return([]*neo4j.Record{
		{
			Keys:   []string{"name", "edition", "versions"},
			Values: []any{"Neo4j Kernel", "enterprise", []any{"5.18.0"}},
		},
		{
			Keys:   []string{"name", "edition", "versions"},
			Values: []any{"Cypher", "enterprise", []any{"5"}},
		},
	}, nil)
	analyticsService := analytics.NewMockService(ctrl)

	t.Run("emits ConnectionInitializedEvent and StartupEvent events on initialize", func(t *testing.T) {
		withFreshStdin(t)

		analyticsService.EXPECT().IsEnabled().Times(1).Return(true)
		analyticsService.EXPECT().NewStartupEvent(config.TransportModeStdio, false, "test-version").Times(1)
		analyticsService.EXPECT().NewConnectionInitializedEvent(analyticsReal.ConnectionEventInfo{
			Neo4jVersion:  "5.18.0",
			Edition:       "enterprise",
			CypherVersion: []string{"5"},
		}).Times(1)
		analyticsService.EXPECT().EmitEvent(gomock.Any()).Times(2) // startup + connection events

		s := server.NewNeo4jMCPServer("test-version", cfg, mockDB, analyticsService)
		if s == nil {
			t.Fatal("NewNeo4jMCPServer() returned nil")
		}
		err := s.Start()

		if err != nil {
			t.Errorf("Start() unexpected error = %v", err)
		}
		_, err = connectInProcessClient(t, context.Background(), s.MCPServer)
		if err != nil {
			t.Fatalf("Expect no error during initialization, got: %s", err.Error())
		}
		// Stop should work without errors
		ctx := context.Background()
		err = s.Stop(ctx)
		if err != nil {
			t.Errorf("Stop() unexpected error = %v", err)
		}
	})
}

// withFreshStdin gives os.Stdin a new file for the duration of the test, restoring the original afterward. 
func withFreshStdin(t *testing.T) {
	t.Helper()

	original := os.Stdin

	devNull, err := os.Open(os.DevNull)
	if err != nil {
		t.Fatalf("failed to open %s: %v", os.DevNull, err)
	}
	os.Stdin = devNull

	t.Cleanup(func() {
		os.Stdin = original
	})
}

// connectInProcessClient connects an MCP client directly to mcpServer over an in-memory transport pair.
func connectInProcessClient(t *testing.T, ctx context.Context, mcpServer *mcp.Server) (*mcp.ClientSession, error) {
	t.Helper()

	serverTransport, clientTransport := mcp.NewInMemoryTransports()
	if _, err := mcpServer.Connect(ctx, serverTransport, nil); err != nil {
		t.Fatalf("unexpected error connecting server transport: %s", err.Error())
	}

	mcpClient := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "1.0.0"}, nil)
	return mcpClient.Connect(ctx, clientTransport, nil)
}
