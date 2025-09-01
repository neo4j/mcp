package tools

import (
	"context"
	"fmt"
	"os"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/neo4j/mcp/internal/config"
	"github.com/neo4j/mcp/internal/database"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

func GetSchemaHandler(deps *ToolDependencies) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleGetSchema(ctx, deps.Driver, deps.Config)
	}
}

func handleGetSchema(ctx context.Context, driver *neo4j.DriverWithContext, config *config.Config) (*mcp.CallToolResult, error) {
	fmt.Fprintf(os.Stderr, "Retrieving Neo4j schema information using APOC\n")

	// Use APOC meta schema query to get comprehensive schema information
	schemaQuery := `
        CALL apoc.meta.schema()
        YIELD value
        UNWIND keys(value) AS key
        WITH key, value[key] AS value
        RETURN key, value { .labels, .properties, .type, .relationships } as value
      `

	// Execute the APOC schema query
	records, err := database.ExecuteReadQuery(ctx, driver, schemaQuery, nil, config.Database)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Error executing APOC schema query: %v", err)), nil
	}

	// Convert records to JSON using the existing utility function
	response, err := database.Neo4jRecordsToJSON(records)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Error formatting schema results: %v", err)), nil
	}

	return mcp.NewToolResultText(response), nil
}
