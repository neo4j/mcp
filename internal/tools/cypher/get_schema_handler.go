package cypher

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/neo4j/mcp/internal/tools"
)

const (
	// schemaQuery is the APOC query used to retrieve comprehensive schema information
	schemaQuery = `
        CALL apoc.meta.schema()
        YIELD value
        UNWIND keys(value) AS key
        WITH key, value[key] AS value
        RETURN key, value { .properties, .type, .relationships } as value
    `
)

// GetSchemaHandler returns a handler function for the get_schema tool
func GetSchemaHandler(deps *tools.ToolDependencies) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleGetSchema(ctx, deps)
	}
}

// handleGetSchema retrieves Neo4j schema information using APOC
func handleGetSchema(ctx context.Context, deps *tools.ToolDependencies) (*mcp.CallToolResult, error) {
	if deps.DBService == nil {
		errMessage := "database service is not initialized"
		deps.Log.Error(errMessage)
		return mcp.NewToolResultError(errMessage), nil
	}

	deps.Log.Info("retrieving schema from the database")

	// Execute the APOC schema query
	records, err := deps.DBService.ExecuteReadQuery(ctx, schemaQuery, nil)
	if err != nil {
		deps.Log.Error("failed to execute schema query", "error", err)
		return mcp.NewToolResultError(err.Error()), nil
	}
	if len(records) == 0 {
		deps.Log.Warn("schema is empty, no data in the database")
		return mcp.NewToolResultText("The get-schema tool executed successfully; however, since the Neo4j instance contains no data, no schema information was returned."), nil
	}
	// Convert records to JSON using the existing utility function
	response, err := deps.DBService.Neo4jRecordsToJSON(records)

	if err != nil {
		deps.Log.Error("failed to format schema results to JSON", "error", err)
		return mcp.NewToolResultError(err.Error()), nil
	}

	deps.Log.Info("successfully retrieved schema")

	return mcp.NewToolResultText(response), nil
}
