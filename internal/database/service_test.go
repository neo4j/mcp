package database_test

import (
	"context"
	"errors"
	"strings"
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

	t.Run("query with parameters - find person by name", func(t *testing.T) {
		mockDriver := mocks.NewMockDriver(ctrl)
		mockSession := mocks.NewMockSession(ctrl)

		// Simulate Neo4j returning a person record
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

		mockDriver.EXPECT().
			NewSession(gomock.Any(), "neo4j").
			Return(mockSession, nil)

		mockSession.EXPECT().
			ExecuteRead(gomock.Any(), gomock.Any()).
			Return(mockRecords, nil)

		mockSession.EXPECT().
			Close(gomock.Any()).
			Return(nil)

		service := database.NewNeo4jServiceWithDriver(mockDriver)
		records, err := service.ExecuteReadQuery(
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

	t.Run("cypher syntax error from tx.Run", func(t *testing.T) {
		mockDriver := mocks.NewMockDriver(ctrl)
		mockSession := mocks.NewMockSession(ctrl)

		mockDriver.EXPECT().
			NewSession(gomock.Any(), "neo4j").
			Return(mockSession, nil)

		// Simulate a Cypher syntax error that would come from tx.Run
		mockSession.EXPECT().
			ExecuteRead(gomock.Any(), gomock.Any()).
			Return(nil, errors.New("Neo.ClientError.Statement.SyntaxError: Invalid syntax at line 1"))

		mockSession.EXPECT().
			Close(gomock.Any()).
			Return(nil)

		service := database.NewNeo4jServiceWithDriver(mockDriver)
		_, err := service.ExecuteReadQuery(
			ctx,
			"MATCH (p:Person WHERE p.name = $name RETURN p", // Invalid Cypher
			map[string]any{"name": "Alice"},
			"neo4j",
		)

		if err == nil {
			t.Errorf("expected cypher syntax error")
		}
		// Verify the error message is propagated
		if err != nil && !strings.Contains(err.Error(), "SyntaxError") {
			t.Errorf("expected syntax error in message, got: %v", err)
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

	t.Run("create node with properties and return it", func(t *testing.T) {
		mockDriver := mocks.NewMockDriver(ctrl)
		mockSession := mocks.NewMockSession(ctrl)

		// Simulate Neo4j returning the created node with generated properties
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

		mockDriver.EXPECT().
			NewSession(gomock.Any(), "neo4j").
			Return(mockSession, nil)

		mockSession.EXPECT().
			ExecuteWrite(gomock.Any(), gomock.Any()).
			Return(mockRecords, nil)

		mockSession.EXPECT().
			Close(gomock.Any()).
			Return(nil)

		service := database.NewNeo4jServiceWithDriver(mockDriver)
		records, err := service.ExecuteWriteQuery(
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

	t.Run("cypher syntax error from tx.Run", func(t *testing.T) {
		mockDriver := mocks.NewMockDriver(ctrl)
		mockSession := mocks.NewMockSession(ctrl)

		mockDriver.EXPECT().
			NewSession(gomock.Any(), "neo4j").
			Return(mockSession, nil)

		// Simulate a Cypher syntax error that would come from tx.Run
		mockSession.EXPECT().
			ExecuteWrite(gomock.Any(), gomock.Any()).
			Return(nil, errors.New("Neo.ClientError.Statement.SyntaxError: Invalid syntax at line 1"))

		mockSession.EXPECT().
			Close(gomock.Any()).
			Return(nil)

		service := database.NewNeo4jServiceWithDriver(mockDriver)
		_, err := service.ExecuteWriteQuery(
			ctx,
			"CREATE (p:Person {name: $name RETURN p", // Invalid Cypher - missing closing brace
			map[string]any{"name": "Alice"},
			"neo4j",
		)

		if err == nil {
			t.Errorf("expected cypher syntax error")
		}
		// Verify the error message is propagated
		if err != nil && !strings.Contains(err.Error(), "SyntaxError") {
			t.Errorf("expected syntax error in message, got: %v", err)
		}
	})
}
