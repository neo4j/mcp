//go:build integration

package integration

import (
	"context"
	"log"
	"os"
	"testing"

	"github.com/neo4j/mcp/internal/config"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

var sharedCfg *config.Config
var sharedDrv neo4j.DriverWithContext

func TestMain(m *testing.M) {
	ctx := context.Background()

	ctr, boltURI, err := createNeo4jContainer(ctx)
	if err != nil {
		log.Fatalf("failed to start shared neo4j container: %v", err)
	}

	username := os.Getenv("NEO4J_USERNAME")
	if username == "" {
		username = "neo4j"
	}
	password := os.Getenv("NEO4J_PASSWORD")
	if password == "" {
		password = "password"
	}

	drv, err := neo4j.NewDriverWithContext(boltURI, neo4j.BasicAuth(username, password, ""))
	if err != nil {
		_ = ctr.Terminate(ctx)
		log.Fatalf("failed to create driver: %v", err)
	}

	if err := waitForConnectivity(ctx, ctr, &drv); err != nil {
		_ = drv.Close(ctx)
		_ = ctr.Terminate(ctx)
		log.Fatalf("failed to verify connectivity: %v", err)
	}

	sharedCfg = &config.Config{URI: boltURI, Username: username, Password: password, Database: "neo4j"}
	sharedDrv = drv

	code := m.Run()

    if err := drv.Close(context.Background()); err != nil {
      log.Printf("Warning: failed to close driver: %v", err)
    }
    if err := ctr.Terminate(context.Background()); err != nil {
      log.Printf("Warning: failed to terminate container: %v", err)
    }

	os.Exit(code)
}
