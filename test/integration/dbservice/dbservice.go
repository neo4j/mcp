package dbservice

import (
	"context"
	"log"

	"github.com/neo4j/mcp/internal/config"
	"github.com/neo4j/mcp/test/integration/containerrunner"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

type dbService struct {
	driver       *neo4j.DriverWithContext
	useContainer bool
}

func NewDBService() *dbService {
	return &dbService{
		driver:       nil,
		useContainer: config.GetEnvWithDefault("MCP_USE_CONTAINER_FOR_INTEGRATION_TESTS", "true") == "true",
	}
}

func (dbs *dbService) Start(ctx context.Context) {
	if dbs.useContainer == true {
		containerrunner.Start(ctx)
	}
}

func (dbs *dbService) Stop(ctx context.Context) {
	if dbs.useContainer == true {
		containerrunner.Close(ctx)
	}
}

func (dbs *dbService) GetDriver() *neo4j.DriverWithContext {
	if dbs.driver != nil {
		return dbs.driver
	}

	if dbs.useContainer == true {
		drv := containerrunner.GetDriver()
		dbs.driver = drv
	} else {
		cfg := &config.Config{
			URI:      config.GetEnvWithDefault("NEO4J_URI", "bolt://localhost:7687"),
			Username: config.GetEnvWithDefault("NEO4J_USERNAME", "neo4j"),
			Password: config.GetEnvWithDefault("NEO4J_PASSWORD", "longerpassword"),
			Database: config.GetEnvWithDefault("NEO4J_DATABASE", "neo4j"),
		}

		drv, err := neo4j.NewDriverWithContext(cfg.URI, neo4j.BasicAuth(cfg.Username, cfg.Password, ""))
		if err != nil {
			log.Fatalf("failed to create driver: %v", err)
		}
		dbs.driver = &drv
	}

	return dbs.driver
}
