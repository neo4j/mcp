//go:build integration || e2e

package dbservice

import (
	"context"
	"log"
	"sync"

	"github.com/neo4j/mcp/internal/config"
	"github.com/neo4j/mcp/test/containerrunner"
	"github.com/neo4j/neo4j-go-driver/v6/neo4j"
)

type dbService struct {
	driver       *neo4j.Driver
	driverOnce   sync.Once // Ensures driver is initialized exactly once
	useContainer bool
}

func NewDBService() *dbService {
	useContainer := config.GetEnvWithDefault("USE_CONTAINER", "true") == "true"
	log.Printf("Testing using container: %t", useContainer)
	return &dbService{
		driver:       nil,
		useContainer: useContainer,
	}
}

func (dbs *dbService) Start(ctx context.Context) {
	if dbs.useContainer {
		containerrunner.Start(ctx)
	}
}

func (dbs *dbService) Stop(ctx context.Context) {
	if dbs.useContainer {
		containerrunner.Close(ctx)
	}
}

func (dbs *dbService) GetDriver() *neo4j.Driver {
	dbs.driverOnce.Do(func() {
		if dbs.useContainer {
			drv := containerrunner.GetDriver()
			dbs.driver = drv
		} else {
			cfg := &config.Config{
				URI:      config.GetEnvWithDefault("NEO4J_URI", "bolt://localhost:7687"),
				Username: config.GetEnvWithDefault("NEO4J_USERNAME", "neo4j"),
				Password: config.GetEnvWithDefault("NEO4J_PASSWORD", "password"),
			}

			drv, err := neo4j.NewDriver(cfg.URI, neo4j.BasicAuth(cfg.Username, cfg.Password, ""))
			if err != nil {
				log.Fatalf("failed to create driver: %v", err)
			}
			dbs.driver = &drv
		}
	})

	return dbs.driver
}

func (dbs *dbService) GetDriverConf() *config.Config {
	if dbs.useContainer == true {
		return containerrunner.GetDriverConf()
	}

	cfg := &config.Config{
		URI:           config.GetEnvWithDefault("NEO4J_URI", "bolt://localhost:7687"),
		Username:      config.GetEnvWithDefault("NEO4J_USERNAME", "neo4j"),
		Password:      config.GetEnvWithDefault("NEO4J_PASSWORD", "password"),
		TransportMode: config.GetTransportModeWithDefault("NEO4J_MCP_TRANSPORT", config.TransportModeStdio),
	}

	return cfg
}
