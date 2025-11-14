package cypher

import (
	"context"
	"log"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/neo4j/mcp/internal/analytics"
	"github.com/neo4j/mcp/internal/database"
	"github.com/neo4j/mcp/internal/tools"
)

const (
	// schemaQuery is the APOC query used to retrieve comprehensive schema information
	schemaQuery = `
        CALL apoc.meta.schema()
        YIELD value
        UNWIND keys(value) as key
        WITH key, value[key] as value
        RETURN key, value { .properties, .type, .relationships } as value
    `
)

// GetSchemaHandler returns a handler function for the get_schema tool
func GetSchemaHandler(deps *tools.ToolDependencies) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleGetSchema(ctx, deps.DBService, deps.AnalyticsService)
	}
}

// handleGetSchema retrieves Neo4j schema information using APOC
func handleGetSchema(ctx context.Context, dbService database.Service, asService analytics.Service) (*mcp.CallToolResult, error) {
	if asService == nil {
		errMessage := "Analytics service is not initialized"
		log.Printf("%s", errMessage)
		return mcp.NewToolResultError(errMessage), nil
	}
	if dbService == nil {
		errMessage := "Database service is not initialized"
		log.Printf("%s", errMessage)
		return mcp.NewToolResultError(errMessage), nil
	}

	asService.EmitEvent(asService.NewToolsEvent("get-schema"))

	// Execute the APOC schema query
	records, err := dbService.ExecuteReadQuery(ctx, schemaQuery, nil)
	if err != nil {
		log.Printf("Failed to execute schema query: %v", err)
		return mcp.NewToolResultError(err.Error()), nil
	}
	if len(records) == 0 {
		return mcp.NewToolResultText("The get-schema tool executed successfully; however, since the Neo4j instance contains no data, no schema information was returned."), nil
	}
	// Convert records to JSON using the existing utility function
	response, err := dbService.Neo4jRecordsToJSON(records)

	if err != nil {
		log.Printf("Failed to format schema results to JSON: %v", err)
		return mcp.NewToolResultError(err.Error()), nil
	}
	return mcp.NewToolResultText(response), nil
}
