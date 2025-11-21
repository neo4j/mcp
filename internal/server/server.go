package server

import (
	"context"
	"fmt"
	"log"
	"log/slog"

	"github.com/mark3labs/mcp-go/server"
	"github.com/neo4j/mcp/internal/analytics"
	"github.com/neo4j/mcp/internal/config"
	"github.com/neo4j/mcp/internal/database"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

// Neo4jMCPServer represents the MCP server instance
type Neo4jMCPServer struct {
	MCPServer    *server.MCPServer
	config       *config.Config
	dbService    database.Service
	version      string
	anService    analytics.Service
	gdsInstalled bool
}

// NewNeo4jMCPServer creates a new MCP server instance
// The config parameter is expected to be already validated
func NewNeo4jMCPServer(version string, cfg *config.Config, dbService database.Service, anService analytics.Service) *Neo4jMCPServer {
	mcpServer := server.NewMCPServer(
		"neo4j-mcp",
		version,
		server.WithToolCapabilities(true),
		server.WithInstructions("This is the Neo4j official MCP server and can provide tool calling to interact with your Neo4j database,"+
			"by inferring the schema with tools like get-schema and executing arbitrary Cypher queries with read-cypher."),
	)

	return &Neo4jMCPServer{
		MCPServer:    mcpServer,
		config:       cfg,
		dbService:    dbService,
		version:      version,
		anService:    anService,
		gdsInstalled: false,
	}
}

// Start initializes and starts the MCP server using stdio transport
func (s *Neo4jMCPServer) Start() error {
	slog.Info("Starting Neo4j MCP Server...")
	err := s.verifyRequirements()
	if err != nil {
		return err
	}

	s.emitStartupEvent()

	// Register tools
	if err := s.registerTools(); err != nil {
		return fmt.Errorf("failed to register tools: %w", err)
	}
	slog.Info("Started Neo4j MCP Server. Now listening for input...")
	// Note: ServeStdio handles its own signal management for graceful shutdown
	return server.ServeStdio(s.MCPServer)
}

// verifyRequirements check the Neo4j requirements:
// - A valid connection with a Neo4j instance.
// - The ability to perform a read query (database name is correctly defined).
// - Required plugin installed: APOC (specifically apoc.meta.schema as it's used for get-schema)
// - In case GDS is not installed a flag is set in the server and tools will be registered accordingly
func (s *Neo4jMCPServer) verifyRequirements() error {
	err := s.dbService.VerifyConnectivity(context.Background())
	if err != nil {
		return fmt.Errorf("impossible to verify connectivity with the Neo4j instance: %w", err)
	}
	// Perform a dummy query to verify correctness of the connection, VerifyConnectivity is not exhaustive.
	records, err := s.dbService.ExecuteReadQuery(context.Background(), "RETURN 1 as first", map[string]any{})

	if err != nil {
		return fmt.Errorf("impossible to verify connectivity with the Neo4j instance: %w", err)
	}
	if len(records) != 1 || len(records[0].Values) != 1 {
		return fmt.Errorf("failed to verify connectivity with the Neo4j instance: unexpected response from test query")
	}
	one, ok := records[0].Values[0].(int64)
	if !ok || one != 1 {
		return fmt.Errorf("failed to verify connectivity with the Neo4j instance: unexpected response from test query")
	}
	// Check for apoc.meta.schema procedure
	checkApocMetaSchemaQuery := "SHOW PROCEDURES YIELD name WHERE name = 'apoc.meta.schema' RETURN count(name) > 0 AS apocMetaSchemaAvailable"

	// Check for apoc.meta.schema availability
	records, err = s.dbService.ExecuteReadQuery(context.Background(), checkApocMetaSchemaQuery, nil)
	if err != nil {
		return fmt.Errorf("failed to check for APOC availability: %w", err)
	}
	if len(records) != 1 || len(records[0].Values) != 1 {
		return fmt.Errorf("failed to verify APOC availability: unexpected response from test query")
	}
	apocMetaSchemaAvailable, ok := records[0].Values[0].(bool)
	if !ok || !apocMetaSchemaAvailable {
		return fmt.Errorf("please ensure the APOC plugin is installed and includes the 'meta' component")
	}
	// Call gds.version procedure to determine if GDS is installed
	records, err = s.dbService.ExecuteReadQuery(context.Background(), "RETURN gds.version() as gdsVersion", nil)
	if err != nil {
		// GDS is optional, so we log a warning and continue, assuming it's not installed.
		log.Print("Impossible to verify GDS installation.")
		s.gdsInstalled = false
		return nil
	}
	if len(records) == 1 && len(records[0].Values) == 1 {
		_, ok := records[0].Values[0].(string)
		if ok {
			s.gdsInstalled = true
		}
	}

	return nil
}

func (s *Neo4jMCPServer) emitStartupEvent() {
	// CALL dbms.components() to collect meta information about the database such version, edition, Cypher version supported
	records, err := s.dbService.ExecuteReadQuery(context.Background(), "CALL dbms.components()", map[string]any{})

	if err != nil {
		slog.Debug("Impossible to collect information using DBMS component, dbms.components() query failed")
		return
	}

	startupInfo := recordsToStartupEventInfo(records, s.version)

	// track startup event
	s.anService.EmitEvent(s.anService.NewStartupEvent(startupInfo))
}

func recordsToStartupEventInfo(records []*neo4j.Record, mcpVersion string) analytics.StartupEventInfo {
	startupInfo := analytics.StartupEventInfo{
		Neo4jVersion:  "not-found",
		Edition:       "not-found",
		CypherVersion: []string{"not-found"},
		McpVersion:    mcpVersion,
	}
	for _, record := range records {
		nameRaw, ok := record.Get("name")
		if !ok {
			slog.Debug("missing 'name' column in dbms.components record")
			continue
		}
		name, ok := nameRaw.(string)
		if !ok {
			slog.Debug("invalid 'name' type in dbms.components record")
			continue
		}

		editionRaw, ok := record.Get("edition")
		if !ok {
			slog.Debug("missing 'edition' column in dbms.components record")
			continue
		}
		edition, ok := editionRaw.(string)
		if !ok {
			slog.Debug("invalid 'edition' type in dbms.components record")
			continue
		}
		versionsRaw, ok := record.Get("versions")
		if !ok {
			slog.Debug("missing 'versions' column in dbms.components record")
			continue
		}
		versions, ok := versionsRaw.([]interface{})
		if !ok {
			slog.Debug("invalid 'versions' type in dbms.components record")
			continue
		}

		switch name {
		case "Neo4j Kernel":
			// versions can be an array, e,g. Cypher can have multiple versions. "Cypher": ["5", "25"]
			if len(versions) > 0 {
				if v, ok := versions[0].(string); ok {
					startupInfo.Neo4jVersion = v
				}
			}

			startupInfo.Edition = edition
		case "Cypher":
			var stringVersions []string
			for _, v := range versions {
				if s, ok := v.(string); ok {
					stringVersions = append(stringVersions, s)
				}
			}

			startupInfo.CypherVersion = stringVersions
		}
	}
	return startupInfo
}

// Stop gracefully stops the server
func (s *Neo4jMCPServer) Stop() error {
	slog.Info("Stopping Neo4j MCP Server...")
	// Currently no cleanup needed - the MCP server handles its own lifecycle
	// Database service cleanup is handled by the caller (main.go)
	return nil
}
