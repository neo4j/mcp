//go:build integration

package dbservice

import (
	"context"
	"log"

	"github.com/neo4j/mcp/internal/config"
	"github.com/neo4j/mcp/test/integration/containerrunner"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

type DBService struct {
	driver       *neo4j.DriverWithContext
	useContainer bool
}

func NewDBService() *DBService {
	return &DBService{
		driver:       nil,
		useContainer: config.GetEnvWithDefault("USE_CONTAINER", "true") == "true",
	}
}

func (dbs *DBService) Start(ctx context.Context) {
	if dbs.useContainer {
		containerrunner.Start(ctx)
	}
}

func (dbs *DBService) Stop(ctx context.Context) {
	if dbs.useContainer {
		containerrunner.Close(ctx)
	}
}

func (dbs *DBService) GetDriver() *neo4j.DriverWithContext {
	if dbs.driver != nil {
		return dbs.driver
	}

	if dbs.useContainer {
		drv := containerrunner.GetDriver()
		dbs.driver = drv
	} else {
		cfg := &config.Config{
			URI:      config.GetEnvWithDefault("NEO4J_URI", "bolt://localhost:7687"),
			Username: config.GetEnvWithDefault("NEO4J_USERNAME", "neo4j"),
			Password: config.GetEnvWithDefault("NEO4J_PASSWORD", "password"),
		}

		drv, err := neo4j.NewDriverWithContext(cfg.URI, neo4j.BasicAuth(cfg.Username, cfg.Password, ""))
		if err != nil {
			log.Fatalf("failed to create driver: %v", err)
		}
		dbs.driver = &drv
	}

	return dbs.driver
}
