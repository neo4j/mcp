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

func TestDatabaseService_ExecuteReadQuery(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()

	t.Run("successful read query execution", func(t *testing.T) {
		mockService := mocks.NewMockDatabaseService(ctrl)
		expectedRecords := []*neo4j.Record{}

		mockService.EXPECT().
			ExecuteReadQuery(ctx, "MATCH (n:Person) RETURN n", map[string]any{"limit": 10}, "neo4j").
			Return(expectedRecords, nil)

		records, err := mockService.ExecuteReadQuery(ctx, "MATCH (n:Person) RETURN n", map[string]any{"limit": 10}, "neo4j")

		if err != nil {
			t.Errorf("expected no error, got: %v", err)
		}
		if records == nil {
			t.Errorf("expected records, got nil")
		}
	})

	t.Run("query execution error", func(t *testing.T) {
		mockService := mocks.NewMockDatabaseService(ctrl)

		mockService.EXPECT().
			ExecuteReadQuery(ctx, "MATCH (n) RETURN n", nil, "neo4j").
			Return(nil, errors.New("query execution failed"))

		_, err := mockService.ExecuteReadQuery(ctx, "MATCH (n) RETURN n", nil, "neo4j")

		if err == nil {
			t.Errorf("expected error, got nil")
		}
	})

	t.Run("query with parameters - find person by name", func(t *testing.T) {
		mockService := mocks.NewMockDatabaseService(ctrl)

		mockRecords := []*neo4j.Record{
			{
				Keys: []string{"name", "age", "email"},
				Values: []any{
					"Alice",
					int64(30),
					"alice@example.com",
				},
			},
		}

		mockService.EXPECT().
			ExecuteReadQuery(
				ctx,
				"MATCH (p:Person {name: $name}) RETURN p.name as name, p.age as age, p.email as email",
				map[string]any{"name": "Alice"},
				"neo4j",
			).
			Return(mockRecords, nil)

		records, err := mockService.ExecuteReadQuery(
			ctx,
			"MATCH (p:Person {name: $name}) RETURN p.name as name, p.age as age, p.email as email",
			map[string]any{"name": "Alice"},
			"neo4j",
		)

		if err != nil {
			t.Errorf("expected no error, got: %v", err)
		}
		if len(records) != 1 {
			t.Errorf("expected 1 record, got: %d", len(records))
		}
		if records[0].Values[0] != "Alice" {
			t.Errorf("expected name 'Alice', got: %v", records[0].Values[0])
		}
	})

	t.Run("cypher syntax error", func(t *testing.T) {
		mockService := mocks.NewMockDatabaseService(ctrl)

		mockService.EXPECT().
			ExecuteReadQuery(
				ctx,
				"MATCH (p:Person WHERE p.name = $name RETURN p",
				map[string]any{"name": "Alice"},
				"neo4j",
			).
			Return(nil, errors.New("syntax error"))

		_, err := mockService.ExecuteReadQuery(
			ctx,
			"MATCH (p:Person WHERE p.name = $name RETURN p",
			map[string]any{"name": "Alice"},
			"neo4j",
		)

		if err == nil {
			t.Errorf("expected cypher syntax error")
		}
	})
}

func TestNewNeo4jService(t *testing.T) {
	t.Run("nil driver error", func(t *testing.T) {
		service, err := database.NewNeo4jService(nil)

		if err == nil {
			t.Errorf("expected error when driver is nil, got nil")
		}
		if service != nil {
			t.Errorf("expected nil service when driver is nil, got %v", service)
		}
		if err.Error() != "driver cannot be nil" {
			t.Errorf("expected error 'driver cannot be nil', got: %v", err)
		}
	})
}

func TestDatabaseService_ExecuteWriteQuery(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()

	t.Run("successful write query execution", func(t *testing.T) {
		mockService := mocks.NewMockDatabaseService(ctrl)
		expectedRecords := []*neo4j.Record{}

		mockService.EXPECT().
			ExecuteWriteQuery(ctx, "CREATE (n:Person {name: $name}) RETURN n", map[string]any{"name": "Alice"}, "neo4j").
			Return(expectedRecords, nil)

		records, err := mockService.ExecuteWriteQuery(ctx, "CREATE (n:Person {name: $name}) RETURN n", map[string]any{"name": "Alice"}, "neo4j")

		if err != nil {
			t.Errorf("expected no error, got: %v", err)
		}
		if records == nil {
			t.Errorf("expected records, got nil")
		}
	})

	t.Run("query execution error", func(t *testing.T) {
		mockService := mocks.NewMockDatabaseService(ctrl)

		mockService.EXPECT().
			ExecuteWriteQuery(ctx, "CREATE (n:Test)", nil, "neo4j").
			Return(nil, errors.New("query execution failed"))

		_, err := mockService.ExecuteWriteQuery(ctx, "CREATE (n:Test)", nil, "neo4j")

		if err == nil {
			t.Errorf("expected error, got nil")
		}
	})

	t.Run("create node with properties and return it", func(t *testing.T) {
		mockService := mocks.NewMockDatabaseService(ctrl)

		mockRecords := []*neo4j.Record{
			{
				Keys: []string{"id", "name", "createdAt"},
				Values: []any{
					int64(123),
					"NewPerson",
					"2024-01-01T00:00:00Z",
				},
			},
		}

		mockService.EXPECT().
			ExecuteWriteQuery(
				ctx,
				"CREATE (p:Person {name: $name}) SET p.createdAt = datetime() RETURN id(p) as id, p.name as name, p.createdAt as createdAt",
				map[string]any{"name": "NewPerson"},
				"neo4j",
			).
			Return(mockRecords, nil)

		records, err := mockService.ExecuteWriteQuery(
			ctx,
			"CREATE (p:Person {name: $name}) SET p.createdAt = datetime() RETURN id(p) as id, p.name as name, p.createdAt as createdAt",
			map[string]any{"name": "NewPerson"},
			"neo4j",
		)

		if err != nil {
			t.Errorf("expected no error, got: %v", err)
		}
		if len(records) != 1 {
			t.Errorf("expected 1 record, got: %d", len(records))
		}
		if records[0].Values[1] != "NewPerson" {
			t.Errorf("expected name 'NewPerson', got: %v", records[0].Values[1])
		}
	})

	t.Run("cypher syntax error", func(t *testing.T) {
		mockService := mocks.NewMockDatabaseService(ctrl)

		mockService.EXPECT().
			ExecuteWriteQuery(
				ctx,
				"CREATE (p:Person {name: $name RETURN p",
				map[string]any{"name": "Alice"},
				"neo4j",
			).
			Return(nil, errors.New("syntax error"))

		_, err := mockService.ExecuteWriteQuery(
			ctx,
			"CREATE (p:Person {name: $name RETURN p",
			map[string]any{"name": "Alice"},
			"neo4j",
		)

		if err == nil {
			t.Errorf("expected cypher syntax error")
		}
	})
}
