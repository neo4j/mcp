package database

import (
	"context"
	"errors"
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
		session, err := factory.NewSession(ctx, "neo4j")
		if err == nil {
			t.Errorf("expected error when driver is nil")
		}
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
			Return(nil, nil)

		service := newNeo4jServiceWithSessionFactory(mockFactory)
		if _, err := service.ExecuteReadQuery(ctx, "MATCH (n) RETURN n", nil, "neo4j"); err == nil {
			t.Errorf("expected error when session is nil")
		}
	})

	t.Run("session factory returns error", func(t *testing.T) {
		mockFactory := mocks.NewMockSessionFactory(ctrl)
		expectedErr := errors.New("failed to create session")
		mockFactory.EXPECT().
			NewSession(gomock.Any(), "neo4j").
			Return(nil, expectedErr)

		service := newNeo4jServiceWithSessionFactory(mockFactory)
		if _, err := service.ExecuteReadQuery(ctx, "MATCH (n) RETURN n", nil, "neo4j"); err == nil {
			t.Errorf("expected error when NewSession returns error")
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
			Return(nil, nil)

		service := newNeo4jServiceWithSessionFactory(mockFactory)
		if _, err := service.ExecuteWriteQuery(ctx, "CREATE (n:Test)", nil, "neo4j"); err == nil {
			t.Errorf("expected error when session is nil")
		}
	})

	t.Run("session factory returns error", func(t *testing.T) {
		mockFactory := mocks.NewMockSessionFactory(ctrl)
		expectedErr := errors.New("failed to create session")
		mockFactory.EXPECT().
			NewSession(gomock.Any(), "neo4j").
			Return(nil, expectedErr)

		service := newNeo4jServiceWithSessionFactory(mockFactory)
		if _, err := service.ExecuteWriteQuery(ctx, "CREATE (n:Test)", nil, "neo4j"); err == nil {
			t.Errorf("expected error when NewSession returns error")
		}
	})
}

func TestNeo4jService_UsesDependencyInjectedSessionFactory(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	mockFactory := mocks.NewMockSessionFactory(ctrl)

	// Set up mock expectation for the behavior we want to verify
	mockFactory.EXPECT().
		NewSession(gomock.Any(), "neo4j").
		Return(nil, errors.New("test error"))

	service := newNeo4jServiceWithSessionFactory(mockFactory)

	// Verify it returns a DatabaseService interface
	if service == nil {
		t.Fatal("expected non-nil service")
	}

	// Test the actual behavior: verify the service uses the injected factory
	_, err := service.ExecuteReadQuery(ctx, "MATCH (n) RETURN n", nil, "neo4j")
	if err == nil {
		t.Fatal("expected service to use the injected session factory and return the mock error")
	}

	// Verify the error message contains our test error, confirming the factory was used
	if err.Error() != "test error" {
		t.Fatalf("expected error from mock factory, got: %v", err)
	}
}
