package tools

import (
	"context"
	"fmt"

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
		return mcp.NewToolResultError("Database service is not initialized"), nil
	}

	if config == nil {
		return mcp.NewToolResultError("Configuration is not provided"), nil
	}

	// Execute the APOC schema query
	records, err := dbService.ExecuteReadQuery(ctx, schemaQuery, nil, config.Database)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to execute schema query: %v", err)), nil
	}

	// Convert records to JSON using the existing utility function
	response, err := dbService.Neo4jRecordsToJSON(records)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to format schema results to JSON: %v", err)), nil
	}

	return mcp.NewToolResultText(response), nil
}
