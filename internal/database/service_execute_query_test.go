package database

import (
	"context"
	"testing"

	"github.com/neo4j/mcp/internal/database/mocks"
	"go.uber.org/mock/gomock"
)

// newNeo4jServiceWithSessionFactory creates a Neo4jService with a custom SessionFactory for testing
func newNeo4jServiceWithSessionFactory(sessionFactory SessionFactory) DatabaseService {
	return &Neo4jService{
		sessionFactory: sessionFactory,
	}
}

func TestNeo4jSessionFactory_NewSession(t *testing.T) {
	ctx := context.Background()

	t.Run("nil driver", func(t *testing.T) {
		factory := &Neo4jSessionFactory{driver: nil}
		session := factory.NewSession(ctx, "neo4j")
		if session != nil {
			t.Errorf("expected nil session when driver is nil, got: %v", session)
		}
	})
}

func TestNeo4jService_ExecuteReadQuery(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()

	t.Run("nil session factory", func(t *testing.T) {
		serviceNil := newNeo4jServiceWithSessionFactory(nil)
		if _, err := serviceNil.ExecuteReadQuery(ctx, "RETURN 1", nil, "neo4j"); err == nil {
			t.Errorf("expected error when session factory is nil")
		}
	})

	t.Run("session factory returns nil session", func(t *testing.T) {
		mockFactory := mocks.NewMockSessionFactory(ctrl)
		mockFactory.EXPECT().
			NewSession(gomock.Any(), "neo4j").
			Return(nil)

		service := newNeo4jServiceWithSessionFactory(mockFactory)
		if _, err := service.ExecuteReadQuery(ctx, "MATCH (n) RETURN n", nil, "neo4j"); err == nil {
			t.Errorf("expected error when session is nil")
		}
	})
}

func TestNeo4jService_ExecuteWriteQuery(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()

	t.Run("nil session factory", func(t *testing.T) {
		serviceNil := newNeo4jServiceWithSessionFactory(nil)
		if _, err := serviceNil.ExecuteWriteQuery(ctx, "CREATE (n:Test)", nil, "neo4j"); err == nil {
			t.Errorf("expected error when session factory is nil")
		}
	})

	t.Run("session factory returns nil session", func(t *testing.T) {
		mockFactory := mocks.NewMockSessionFactory(ctrl)
		mockFactory.EXPECT().
			NewSession(gomock.Any(), "neo4j").
			Return(nil)

		service := newNeo4jServiceWithSessionFactory(mockFactory)
		if _, err := service.ExecuteWriteQuery(ctx, "CREATE (n:Test)", nil, "neo4j"); err == nil {
			t.Errorf("expected error when session is nil")
		}
	})
}

func TestNewNeo4jServiceWithSessionFactory(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockFactory := mocks.NewMockSessionFactory(ctrl)
	service := newNeo4jServiceWithSessionFactory(mockFactory)

	// Verify it returns a DatabaseService interface
	if service == nil {
		t.Fatal("expected non-nil service")
	}

	// Verify it's actually a Neo4jService underneath
	neo4jService, ok := service.(*Neo4jService)
	if !ok {
		t.Fatal("expected Neo4jService type")
	}

	if neo4jService.sessionFactory != mockFactory {
		t.Fatal("expected session factory to be set correctly")
	}
}
