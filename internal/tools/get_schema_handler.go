package tools

import (
	"context"
	"log"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/neo4j/mcp/internal/config"
	"github.com/neo4j/mcp/internal/database"
)

const (
	// schemaQuery is the APOC query used to retrieve comprehensive schema information
	schemaQuery = `
        CALL apoc.meta.schema()
        YIELD value
        UNWIND keys(value) AS key
        WITH key, value[key] AS value
        RETURN key, value { .labels, .properties, .type, .relationships } as value
    `
)

// GetSchemaHandler returns a handler function for the get_schema tool
func GetSchemaHandler(deps *ToolDependencies) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleGetSchema(ctx, deps.DBService, deps.Config)
	}
}

// handleGetSchema retrieves Neo4j schema information using APOC
func handleGetSchema(ctx context.Context, dbService database.DatabaseService, config *config.Config) (*mcp.CallToolResult, error) {
	if dbService == nil {
		errMessage := "Database service is not initialized"
		log.Printf("%s", errMessage)
		return mcp.NewToolResultError(errMessage), nil
	}

	if config == nil {
		errMessage := "Configuration is not provided"
		log.Printf("%s", errMessage)
		return mcp.NewToolResultError(errMessage), nil
	}

	// Execute the APOC schema query
	records, err := dbService.ExecuteReadQuery(ctx, schemaQuery, nil, config.Database)
	if err != nil {
		log.Printf("Failed to execute schema query: %v", err)
		return mcp.NewToolResultError(err.Error()), nil
	}

	// Convert records to JSON using the existing utility function
	response, err := dbService.Neo4jRecordsToJSON(records)
	if err != nil {
		log.Printf("Failed to format schema results to JSON: %v", err)
		return mcp.NewToolResultError(err.Error()), nil
	}

	return mcp.NewToolResultText(response), nil
}
