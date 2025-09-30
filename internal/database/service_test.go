package database_test

import (
	"context"
	"errors"
	"testing"

	"github.com/neo4j/mcp/internal/database"
	"github.com/neo4j/mcp/internal/database/mocks"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"go.uber.org/mock/gomock"
)

func TestNeo4jService_ExecuteReadQuery(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()

	t.Run("nil driver", func(t *testing.T) {
		service := database.NewNeo4jService(nil)
		_, err := service.ExecuteReadQuery(ctx, "RETURN 1", nil, "neo4j")
		if err == nil {
			t.Errorf("expected error when driver is nil")
		}
	})

	t.Run("nil driver interface", func(t *testing.T) {
		service := database.NewNeo4jServiceWithDriver(nil)
		_, err := service.ExecuteReadQuery(ctx, "RETURN 1", nil, "neo4j")
		if err == nil {
			t.Errorf("expected error when driver is nil")
		}
	})

	t.Run("session creation error", func(t *testing.T) {
		mockDriver := mocks.NewMockDriver(ctrl)
		mockDriver.EXPECT().
			NewSession(gomock.Any(), "neo4j").
			Return(nil, errors.New("failed to create session"))

		service := database.NewNeo4jServiceWithDriver(mockDriver)
		_, err := service.ExecuteReadQuery(ctx, "MATCH (n) RETURN n", nil, "neo4j")
		if err == nil {
			t.Errorf("expected error when session creation fails")
		}
	})

	t.Run("successful read query execution", func(t *testing.T) {
		mockDriver := mocks.NewMockDriver(ctrl)
		mockSession := mocks.NewMockSession(ctrl)
		expectedRecords := []*neo4j.Record{}

		mockDriver.EXPECT().
			NewSession(gomock.Any(), "neo4j").
			Return(mockSession, nil)

		mockSession.EXPECT().
			ExecuteRead(gomock.Any(), gomock.Any()).
			Return(expectedRecords, nil)

		mockSession.EXPECT().
			Close(gomock.Any()).
			Return(nil)

		service := database.NewNeo4jServiceWithDriver(mockDriver)
		records, err := service.ExecuteReadQuery(ctx, "MATCH (n:Person) RETURN n", map[string]any{"limit": 10}, "neo4j")

		if err != nil {
			t.Errorf("expected no error, got: %v", err)
		}
		if records == nil {
			t.Errorf("expected records, got nil")
		}
	})

	t.Run("transaction error", func(t *testing.T) {
		mockDriver := mocks.NewMockDriver(ctrl)
		mockSession := mocks.NewMockSession(ctrl)

		mockDriver.EXPECT().
			NewSession(gomock.Any(), "neo4j").
			Return(mockSession, nil)

		mockSession.EXPECT().
			ExecuteRead(gomock.Any(), gomock.Any()).
			Return(nil, errors.New("transaction failed"))

		mockSession.EXPECT().
			Close(gomock.Any()).
			Return(nil)

		service := database.NewNeo4jServiceWithDriver(mockDriver)
		_, err := service.ExecuteReadQuery(ctx, "MATCH (n) RETURN n", nil, "neo4j")

		if err == nil {
			t.Errorf("expected error, got nil")
		}
	})

	t.Run("unexpected result type", func(t *testing.T) {
		mockDriver := mocks.NewMockDriver(ctrl)
		mockSession := mocks.NewMockSession(ctrl)

		mockDriver.EXPECT().
			NewSession(gomock.Any(), "neo4j").
			Return(mockSession, nil)

		mockSession.EXPECT().
			ExecuteRead(gomock.Any(), gomock.Any()).
			Return("unexpected string", nil)

		mockSession.EXPECT().
			Close(gomock.Any()).
			Return(nil)

		service := database.NewNeo4jServiceWithDriver(mockDriver)
		_, err := service.ExecuteReadQuery(ctx, "MATCH (n) RETURN n", nil, "neo4j")

		if err == nil {
			t.Errorf("expected error for unexpected result type")
		}
	})
}

func TestNeo4jService_ExecuteWriteQuery(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()

	t.Run("nil driver", func(t *testing.T) {
		service := database.NewNeo4jService(nil)
		_, err := service.ExecuteWriteQuery(ctx, "CREATE (n:Test)", nil, "neo4j")
		if err == nil {
			t.Errorf("expected error when driver is nil")
		}
	})

	t.Run("nil driver interface", func(t *testing.T) {
		service := database.NewNeo4jServiceWithDriver(nil)
		_, err := service.ExecuteWriteQuery(ctx, "CREATE (n:Test)", nil, "neo4j")
		if err == nil {
			t.Errorf("expected error when driver is nil")
		}
	})

	t.Run("session creation error", func(t *testing.T) {
		mockDriver := mocks.NewMockDriver(ctrl)
		mockDriver.EXPECT().
			NewSession(gomock.Any(), "neo4j").
			Return(nil, errors.New("failed to create session"))

		service := database.NewNeo4jServiceWithDriver(mockDriver)
		_, err := service.ExecuteWriteQuery(ctx, "CREATE (n:Test)", nil, "neo4j")
		if err == nil {
			t.Errorf("expected error when session creation fails")
		}
	})

	t.Run("successful write query execution", func(t *testing.T) {
		mockDriver := mocks.NewMockDriver(ctrl)
		mockSession := mocks.NewMockSession(ctrl)
		expectedRecords := []*neo4j.Record{}

		mockDriver.EXPECT().
			NewSession(gomock.Any(), "neo4j").
			Return(mockSession, nil)

		mockSession.EXPECT().
			ExecuteWrite(gomock.Any(), gomock.Any()).
			Return(expectedRecords, nil)

		mockSession.EXPECT().
			Close(gomock.Any()).
			Return(nil)

		service := database.NewNeo4jServiceWithDriver(mockDriver)
		records, err := service.ExecuteWriteQuery(ctx, "CREATE (n:Person {name: $name}) RETURN n", map[string]any{"name": "Alice"}, "neo4j")

		if err != nil {
			t.Errorf("expected no error, got: %v", err)
		}
		if records == nil {
			t.Errorf("expected records, got nil")
		}
	})

	t.Run("transaction error", func(t *testing.T) {
		mockDriver := mocks.NewMockDriver(ctrl)
		mockSession := mocks.NewMockSession(ctrl)

		mockDriver.EXPECT().
			NewSession(gomock.Any(), "neo4j").
			Return(mockSession, nil)

		mockSession.EXPECT().
			ExecuteWrite(gomock.Any(), gomock.Any()).
			Return(nil, errors.New("transaction failed"))

		mockSession.EXPECT().
			Close(gomock.Any()).
			Return(nil)

		service := database.NewNeo4jServiceWithDriver(mockDriver)
		_, err := service.ExecuteWriteQuery(ctx, "CREATE (n:Test)", nil, "neo4j")

		if err == nil {
			t.Errorf("expected error, got nil")
		}
	})

	t.Run("unexpected result type", func(t *testing.T) {
		mockDriver := mocks.NewMockDriver(ctrl)
		mockSession := mocks.NewMockSession(ctrl)

		mockDriver.EXPECT().
			NewSession(gomock.Any(), "neo4j").
			Return(mockSession, nil)

		mockSession.EXPECT().
			ExecuteWrite(gomock.Any(), gomock.Any()).
			Return("unexpected string", nil)

		mockSession.EXPECT().
			Close(gomock.Any()).
			Return(nil)

		service := database.NewNeo4jServiceWithDriver(mockDriver)
		_, err := service.ExecuteWriteQuery(ctx, "CREATE (n:Test)", nil, "neo4j")

		if err == nil {
			t.Errorf("expected error for unexpected result type")
		}
	})
}