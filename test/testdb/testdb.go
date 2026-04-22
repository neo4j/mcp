// Copyright (c) "Neo4j"
// Neo4j Sweden AB [http://neo4j.com]

//go:build integration || e2e

package testdb

import (
	"context"
	"log"
	"sync"

	"github.com/neo4j/mcp/internal/config"
	"github.com/neo4j/mcp/test/containerrunner"
	"github.com/neo4j/neo4j-go-driver/v6/neo4j"
)

var (
	instance *TestDB
	once     sync.Once
)

func GetInstance() *TestDB {
	once.Do(func() {
		instance = new()
	})
	return instance
}

type TestDB struct {
	driver       *neo4j.Driver
	driverOnce   sync.Once // Ensures driver is initialized exactly once
	useContainer bool
}

func new() *TestDB {
	useContainer := config.GetEnvWithDefault("USE_CONTAINER", "true") == "true"
	log.Printf("Testing using container: %t", useContainer)
	return &TestDB{
		driver:       nil,
		useContainer: useContainer,
	}
}

func (dbs *TestDB) Start(ctx context.Context) {
	if dbs.useContainer {
		containerrunner.Start(ctx)
	}
}

func (dbs *TestDB) Stop(ctx context.Context) {
	if dbs.useContainer {
		containerrunner.Close(ctx)
	}
}

func (dbs *TestDB) GetDriver() *neo4j.Driver {
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

func (dbs *TestDB) GetDriverConf() *config.Config {
	if dbs.useContainer {
		return containerrunner.GetDriverConf()
	}

	transportMode := config.GetTransportModeWithDefault("NEO4J_MCP_TRANSPORT", config.TransportModeStdio)
	cfg := &config.Config{
		URI:           config.GetEnvWithDefault("NEO4J_URI", "bolt://localhost:7687"),
		Username:      config.GetEnvWithDefault("NEO4J_USERNAME", "neo4j"),
		Password:      config.GetEnvWithDefault("NEO4J_PASSWORD", "password"),
		TransportMode: transportMode,
	}
	if transportMode == config.TransportModeStdio {
		cfg.Database = config.GetEnvWithDefault("NEO4J_DATABASE", "neo4j")
	}

	return cfg
}
