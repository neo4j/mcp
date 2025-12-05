package gds_test

import (
	"context"
	"errors"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	analytics "github.com/neo4j/mcp/internal/analytics/mocks"
	"github.com/neo4j/mcp/internal/config"
	db "github.com/neo4j/mcp/internal/database/mocks"
	"github.com/neo4j/mcp/internal/tools"
	"github.com/neo4j/mcp/internal/tools/gds"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"go.uber.org/mock/gomock"
)

func TestListGdsProceduresHandler(t *testing.T) {
	ctrl := gomock.NewController(t)
	analyticsService := analytics.NewMockService(ctrl)
	analyticsService.EXPECT().NewToolsEvent("list-gds-procedures").AnyTimes()
	analyticsService.EXPECT().EmitEvent(gomock.Any()).AnyTimes()
	defer ctrl.Finish()

	t.Run("successful list-gds-procedures", func(t *testing.T) {
		mockDB := db.NewMockService(ctrl)
		mockDB.EXPECT().
			ExecuteReadQuery(gomock.Any(), gomock.Any(), gomock.Nil()).
			Return([]*neo4j.Record{}, nil)
		mockDB.EXPECT().
			Neo4jRecordsToJSON(gomock.Any()).
			Return("", nil)

		deps := &tools.ToolDependencies{
			DBService:        mockDB,
			AnalyticsService: analyticsService,
			Config: &config.Config{
				TransportMode: config.TransportModeStdio,
			},
		}

		handler := gds.ListGdsProceduresHandler(deps)
		request := mcp.CallToolRequest{}

		result, err := handler(context.Background(), request)

		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
		if result == nil || result.IsError {
			t.Error("Expected success result")
		}
	})

	t.Run("nil database service", func(t *testing.T) {
		deps := &tools.ToolDependencies{
			DBService:        nil,
			AnalyticsService: analyticsService,
			Config: &config.Config{
				TransportMode: config.TransportModeStdio,
			},
		}

		handler := gds.ListGdsProceduresHandler(deps)
		request := mcp.CallToolRequest{}

		result, err := handler(context.Background(), request)

		if err != nil {
			t.Errorf("Expected no error from handler, got: %v", err)
		}
		if result == nil || !result.IsError {
			t.Error("Expected error result for nil database service")
		}
	})
	t.Run("nil analytics service", func(t *testing.T) {
		mockDB := db.NewMockService(ctrl)
		deps := &tools.ToolDependencies{
			DBService:        mockDB,
			AnalyticsService: nil,
			Config: &config.Config{
				TransportMode: config.TransportModeStdio,
			},
		}

		handler := gds.ListGdsProceduresHandler(deps)
		result, err := handler(context.Background(), mcp.CallToolRequest{})

		if err != nil {
			t.Errorf("Expected no error from handler, got: %v", err)
		}
		if result == nil || !result.IsError {
			t.Error("Expected error result for nil analytics service")
		}
	})

	t.Run("database query execution failure", func(t *testing.T) {
		mockDB := db.NewMockService(ctrl)
		mockDB.EXPECT().
			ExecuteReadQuery(gomock.Any(), gomock.Any(), gomock.Nil()).
			Return(nil, errors.New("Invalid Cypher"))

		deps := &tools.ToolDependencies{
			DBService:        mockDB,
			AnalyticsService: analyticsService,
			Config: &config.Config{
				TransportMode: config.TransportModeStdio,
			},
		}

		handler := gds.ListGdsProceduresHandler(deps)
		request := mcp.CallToolRequest{}

		result, err := handler(context.Background(), request)

		if err != nil {
			t.Errorf("Expected no error from handler, got: %v", err)
		}
		if result == nil || !result.IsError {
			t.Error("Expected error result for query execution failure")
		}
	})

	t.Run("JSON formatting failure", func(t *testing.T) {
		mockDB := db.NewMockService(ctrl)

		mockDB.EXPECT().
			ExecuteReadQuery(gomock.Any(), gomock.Any(), gomock.Nil()).
			Return([]*neo4j.Record{}, nil)
		mockDB.EXPECT().
			Neo4jRecordsToJSON(gomock.Any()).
			Return("", errors.New("JSON marshaling failed"))

		deps := &tools.ToolDependencies{
			DBService:        mockDB,
			AnalyticsService: analyticsService,
			Config: &config.Config{
				TransportMode: config.TransportModeStdio,
			},
		}

		handler := gds.ListGdsProceduresHandler(deps)
		request := mcp.CallToolRequest{}

		result, err := handler(context.Background(), request)

		if err != nil {
			t.Errorf("Expected no error from handler, got: %v", err)
		}
		if result == nil || !result.IsError {
			t.Error("Expected error result for JSON formatting failure")
		}
	})
}
