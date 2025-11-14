//go:build integration

package integration

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/neo4j/mcp/internal/config"
	"github.com/neo4j/mcp/internal/database"
	"github.com/neo4j/mcp/internal/server"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

func TestServerLifecycle(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name         string
		config       *config.Config
		expectError  bool
		startTimeout time.Duration
	}{
		{
			name: "Neo4jMCPServer should correctly start",
			config: &config.Config{
				URI:      "bolt://localhost:7687",
				Username: "neo4j",
				Password: "password",
				Database: "neo4j",
			},
			startTimeout: 1 * time.Second,
			expectError:  false,
		},
		{
			name: "Neo4jMCPServer should fail to start: invalid host",
			config: &config.Config{
				URI:      "bolt://not-a-valid-host:7687",
				Username: "neo4j",
				Password: "password",
				Database: "neo4j",
			},
			startTimeout: 4 * time.Second,
			expectError:  true,
		},
		{
			name: "Neo4jMCPServer should fail to start: invalid database name",
			config: &config.Config{
				URI:      "bolt://localhost:7687",
				Username: "neo4j",
				Password: "password",
				Database: "not-a-valid-db-name",
			},
			startTimeout: 4 * time.Second,
			expectError:  true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Initialize Neo4j driver
			driver, err := neo4j.NewDriverWithContext(tc.config.URI, neo4j.BasicAuth(tc.config.Username, tc.config.Password, ""))
			if err != nil {
				t.Fatalf("failed to create Neo4j driver: %s", err.Error())
			}

			ctx := context.Background()
			defer func() {
				if err := driver.Close(ctx); err != nil {
					t.Fatalf("error closing driver: %s", err.Error())
				}
			}()

			dbService, err := database.NewNeo4jService(driver, tc.config.Database)
			if err != nil {
				t.Fatalf("failed to create database service: %v", err)
				return
			}

			s := server.NewNeo4jMCPServer("test-version", tc.config, dbService)

			if s == nil {
				t.Fatal("the NewNeo4jMCPServer() returned nil")
			}

			ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
			defer cancel()

			var wg sync.WaitGroup
			wg.Add(1)

			var startErr error
			go func() {
				defer wg.Done()
				startErr = s.Start()
			}()

			for {
				select {
				case <-ctx.Done():
					if tc.expectError {
						if startErr == nil {
							t.Fatal("expected an error but got nil")
						}
					} else {
						if startErr != nil {
							t.Fatalf("Start returned an unexpected error: %s", startErr.Error())
						}
					}
					return
				default:
					time.Sleep(50 * time.Millisecond)
				}
			}
		})
	}

	t.Run("server stop should return no errors", func(t *testing.T) {
		cfg := &config.Config{
			URI:      "bolt://localhost:7687",
			Username: "neo4j",
			Password: "password",
			Database: "neo4j",
		}

		driver, err := neo4j.NewDriverWithContext(cfg.URI, neo4j.BasicAuth(cfg.Username, cfg.Password, ""))
		if err != nil {
			t.Fatalf("failed to create Neo4j driver: %s", err.Error())
		}

		ctx := context.Background()
		defer func() {
			if err := driver.Close(ctx); err != nil {
				t.Fatalf("error closing driver: %s", err.Error())
			}
		}()

		dbService, err := database.NewNeo4jService(driver, cfg.Database)
		if err != nil {
			t.Fatalf("failed to create database service: %v", err)
		}

		s := server.NewNeo4jMCPServer("test-version", cfg, dbService)
		if s == nil {
			t.Fatal("NewNeo4jMCPServer() returned nil")
		}

		var wg sync.WaitGroup
		wg.Add(1)

		var startErr error
		go func() {
			defer wg.Done()
			startErr = s.Start()
		}()

		// Give the server a moment to start
		time.Sleep(1 * time.Second)

		if startErr != nil {
			t.Fatalf("Start() returned an unexpected error after stop: %v", startErr)
		}
		if err := s.Stop(); err != nil {
			t.Fatalf("Stop() returned an unexpected error: %v", err)
		}
	})
}
