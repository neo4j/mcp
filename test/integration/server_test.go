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

func TestServerCorrectlyStart(t *testing.T) {
	t.Parallel()
	t.Run("Neo4jMCPServer should correctly start", func(t *testing.T) {
		cfg := &config.Config{
			URI:      "bolt://localhost:7687",
			Username: "neo4j",
			Password: "password",
			Database: "neo4j",
		}

		// Initialize Neo4j driver
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
			return
		}

		s := server.NewNeo4jMCPServer("test-version", cfg, dbService)

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
				if startErr != nil {
					t.Fatalf("Start returned an unexpected error: %s", startErr.Error())
				}
				return
			default:
				time.Sleep(50 * time.Millisecond)
			}
		}
	})

	t.Run("Neo4jMCPServer should fail to start: invalid host", func(t *testing.T) {
		cfg := &config.Config{
			URI:      "bolt://not-a-valid-host:7687",
			Username: "neo4j",
			Password: "password",
			Database: "neo4j",
		}
		// Initialize Neo4j driver
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
			return
		}

		s := server.NewNeo4jMCPServer("test-version", cfg, dbService)

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
				if startErr == nil {
					t.Fatal("expected VerifyRequirements error")
				}
				return
			default:
				time.Sleep(50 * time.Millisecond)
			}
		}
	})

	t.Run("Neo4jMCPServer should fail to start: invalid database name", func(t *testing.T) {
		cfg := &config.Config{
			URI:      "bolt://localhost:7687",
			Username: "neo4j",
			Password: "password",
			Database: "not-a-valid-db-name",
		}
		// Initialize Neo4j driver
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
			return
		}

		s := server.NewNeo4jMCPServer("test-version", cfg, dbService)

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
				if startErr == nil {
					t.Fatal("expected VerifyRequirements error")
				}
				return
			default:
				time.Sleep(50 * time.Millisecond)
			}
		}
	})
}
