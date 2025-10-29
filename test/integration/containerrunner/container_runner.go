//go:build integration

package containerrunner

import (
	"context"
	"fmt"
	"io"
	"log"
	"sync"
	"time"

	"github.com/neo4j/mcp/internal/config"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

var (
	container testcontainers.Container
	driver    *neo4j.DriverWithContext
	once      sync.Once
)

// Start initializes shared resources for integration tests
func Start(ctx context.Context) {
	once.Do(func() {
		startOnce(ctx)
	})
}

// GetDriver get a driver associated with the instance created
func GetDriver() *neo4j.DriverWithContext {
	if driver == nil {
		log.Fatal("driver is not initialized")
	}
	return driver
}

// startOnce start the testcontainer imaged
func startOnce(ctx context.Context) {
	ctr, boltURI, err := createNeo4jContainer(ctx)
	if err != nil {
		log.Fatalf("failed to start shared neo4j container: %v", err)
	}
	container = ctr

	cfg := &config.Config{
		URI:      boltURI,
		Username: config.GetEnvWithDefault("NEO4J_USERNAME", "neo4j"),
		Password: config.GetEnvWithDefault("NEO4J_PASSWORD", "password"),
		Database: config.GetEnvWithDefault("NEO4J_DATABASE", "neo4j"),
	}

	drv, err := neo4j.NewDriverWithContext(cfg.URI, neo4j.BasicAuth(cfg.Username, cfg.Password, ""))
	driver = &drv
	if err != nil {
		_ = ctr.Terminate(ctx)
		log.Fatalf("failed to create driver: %v", err)
	}

	if err := waitForConnectivity(ctx, ctr); err != nil {
		Close(ctx)
		log.Fatalf("failed to verify connectivity: %v", err)
	}

}

// Close cleans up shared resources used in integration tests
func Close(ctx context.Context) {
	if err := (*driver).Close(ctx); err != nil {
		log.Printf("Warning: failed to close driver: %v", err)
	}
	if err := container.Terminate(ctx); err != nil {
		log.Printf("Warning: failed to terminate container: %v", err)
	}
}

// createNeo4jContainer starts a Neo4j container for testing
func createNeo4jContainer(ctx context.Context) (testcontainers.Container, string, error) {
	req := testcontainers.ContainerRequest{
		Image:        config.GetEnvWithDefault("NEO4J_IMAGE", "neo4j:5.24.2-community"),
		ExposedPorts: []string{"7687/tcp"},
		Env: map[string]string{
			"NEO4J_AUTH":        fmt.Sprintf("%s/%s", config.GetEnvWithDefault("NEO4J_USERNAME", "neo4j"), config.GetEnvWithDefault("NEO4J_PASSWORD", "password")),
			"NEO4JLABS_PLUGINS": config.GetEnvWithDefault("NEO4JLABS_PLUGINS", `["apoc","graph-data-science"]`),
		},
		WaitingFor: wait.ForListeningPort("7687/tcp").WithStartupTimeout(119 * time.Second),
	}

	ctr, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return nil, "", err
	}

	host, err := ctr.Host(ctx)
	if err != nil {
		_ = ctr.Terminate(ctx)
		return nil, "", err
	}

	port, err := ctr.MappedPort(ctx, "7687/tcp")
	if err != nil {
		_ = ctr.Terminate(ctx)
		return nil, "", err
	}

	boltURI := fmt.Sprintf("bolt://%s:%s", host, port.Port())

	return ctr, boltURI, nil
}

// waitForConnectivity waits for Neo4j connectivity with exponential backoff.
func waitForConnectivity(ctx context.Context, ctr testcontainers.Container) error {
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	backoff := 100 * time.Millisecond
	maxBackoff := 2 * time.Second

	var lastErr error
	for {
		err := (*driver).VerifyConnectivity(ctx)
		if err == nil {
			return nil
		}
		lastErr = err

		if ctx.Err() != nil {
			break
		}

		time.Sleep(backoff)
		backoff *= 2
		if backoff > maxBackoff {
			backoff = maxBackoff
		}
	}

	var logs string
	if ctr != nil {
		rc, err := ctr.Logs(context.Background())
		if err == nil && rc != nil {
			b, rerr := io.ReadAll(rc)
			_ = rc.Close()
			if rerr == nil {
				logs = string(b)
			}
		}
	}

	if logs != "" {
		return fmt.Errorf("neo4j connectivity not ready: %v\ncontainer logs:\n%s", lastErr, logs)
	}
	return fmt.Errorf("neo4j connectivity not ready: %v", lastErr)
}
