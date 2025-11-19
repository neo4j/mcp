package server_test

import (
	"fmt"
	"testing"

	analytics "github.com/neo4j/mcp/internal/analytics/mocks"
	"github.com/neo4j/mcp/internal/config"
	db "github.com/neo4j/mcp/internal/database/mocks"
	"github.com/neo4j/mcp/internal/server"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"go.uber.org/mock/gomock"
)

func TestNewNeo4jMCPServer(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	cfg := &config.Config{
		URI:      "bolt://test-host:7687",
		Username: "neo4j",
		Password: "password",
		Database: "neo4j",
	}
	analyticsService := analytics.NewMockService(ctrl)
	analyticsService.EXPECT().EmitEvent(gomock.Any()).AnyTimes()
	analyticsService.EXPECT().NewStartupEvent().AnyTimes()

	t.Run("starts server successfully", func(t *testing.T) {
		mockDB := db.NewMockService(ctrl)
		mockDB.EXPECT().VerifyConnectivity(gomock.Any()).Times(1)
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

		s := server.NewNeo4jMCPServer("test-version", cfg, mockDB, analyticsService)

		if s == nil {
			t.Errorf("NewNeo4jMCPServer() expected non-nil server, got nil")
		}

		err := s.Start()
		if err != nil {
			t.Errorf("Start() unexpected error = %v", err)
		}
	})

	t.Run("starts server should fails when no connection can be established", func(t *testing.T) {
		mockDB := db.NewMockService(ctrl)
		mockDB.EXPECT().VerifyConnectivity(gomock.Any()).Times(1).Return(fmt.Errorf("connection error"))
		s := server.NewNeo4jMCPServer("test-version", cfg, mockDB, analyticsService)

		if s == nil {
			t.Errorf("NewNeo4jMCPServer() expected non-nil server, got nil")
		}

		err := s.Start()
		if err == nil {
			t.Errorf("Start() expected an error, got nil")
		}
	})
	t.Run("starts server should fail when test query returns unexpected result", func(t *testing.T) {
		mockDB := db.NewMockService(ctrl)
		mockDB.EXPECT().VerifyConnectivity(gomock.Any()).Return(nil)
		mockDB.EXPECT().ExecuteReadQuery(gomock.Any(), "RETURN 1 as first", gomock.Any()).Times(1).Return([]*neo4j.Record{
			{
				Keys:   []string{"first"},
				Values: []any{int64(2)}, // Return a value other than 1
			},
		}, nil)
		s := server.NewNeo4jMCPServer("test-version", cfg, mockDB, analyticsService)

		if s == nil {
			t.Errorf("NewNeo4jMCPServer() expected non-nil server, got nil")
		}

		err := s.Start()
		if err == nil {
			t.Errorf("Start() expected an error for unexpected query result, got nil")
		}
	})

	t.Run("server creates successfully with all required components", func(t *testing.T) {
		mockDB := db.NewMockService(ctrl)
		mockDB.EXPECT().VerifyConnectivity(gomock.Any()).Times(1)
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

		s := server.NewNeo4jMCPServer("test-version", cfg, mockDB, analyticsService)

		if s == nil {
			t.Fatal("NewNeo4jMCPServer() returned nil")
		}

		// Start should work without errors
		err := s.Start()
		if err != nil {
			t.Errorf("Start() unexpected error = %v", err)
		}

		// Stop should work without errors
		err = s.Stop()
		if err != nil {
			t.Errorf("Stop() unexpected error = %v", err)
		}
	})

	t.Run("starts server successfully if GDS is not found", func(t *testing.T) {
		mockDB := db.NewMockService(ctrl)
		mockDB.EXPECT().VerifyConnectivity(gomock.Any()).Times(1)
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

		s := server.NewNeo4jMCPServer("test-version", cfg, mockDB, analyticsService)

		if s == nil {
			t.Errorf("NewNeo4jMCPServer() expected non-nil server, got nil")
		}
		err := s.Start()
		if err != nil {
			t.Errorf("Start() unexpected error = %v", err)
		}
	})

	t.Run("stops server successfully", func(t *testing.T) {
		mockDB := db.NewMockService(ctrl)
		mockDB.EXPECT().VerifyConnectivity(gomock.Any()).Times(1)
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

		s := server.NewNeo4jMCPServer("test-version", cfg, mockDB, analyticsService)

		if s == nil {
			t.Errorf("NewNeo4jMCPServer() expected non-nil server, got nil")
		}

		err := s.Start()
		if err != nil {
			t.Errorf("Start() unexpected error = %v", err)
		}
	})

}

func TestNewNeo4jMCPServerEvents(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	cfg := &config.Config{
		URI:      "bolt://test-host:7687",
		Username: "neo4j",
		Password: "password",
		Database: "neo4j",
	}

	mockDB := db.NewMockService(ctrl)
	mockDB.EXPECT().VerifyConnectivity(gomock.Any()).AnyTimes()
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
	analyticsService := analytics.NewMockService(ctrl)

	t.Run("emits startup and OSInfoEvent and StartupEvent events on start", func(t *testing.T) {
		analyticsService.EXPECT().NewStartupEvent().Times(1)
		analyticsService.EXPECT().EmitEvent(gomock.Any()).Times(1)

		s := server.NewNeo4jMCPServer("test-version", cfg, mockDB, analyticsService)
		if s == nil {
			t.Fatal("NewNeo4jMCPServer() returned nil")
		}
		err := s.Start()
		if err != nil {
			t.Errorf("Start() unexpected error = %v", err)
		}
		// Stop should work without errors
		err = s.Stop()
		if err != nil {
			t.Errorf("Stop() unexpected error = %v", err)
		}
	})
}
