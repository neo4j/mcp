package tools

import (
	"context"
	"log"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/neo4j/mcp/internal/config"
	"github.com/neo4j/mcp/internal/database"
)

func ListOfGDSFunctionsHandler(deps *ToolDependencies) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleReturnListOfGDSFunctions(ctx, request, deps.DBService, deps.Config)
	}
}

func handleReturnListOfGDSFunctions(ctx context.Context, request mcp.CallToolRequest, dbService database.Service, config *config.Config) (*mcp.CallToolResult, error) {

	// Gets the list of available GDS functions
	query := `SHOW PROCEDURES YIELD * 
	WHERE name STARTS WITH 'gds.' 
	AND NOT name CONTAINS '.estimate'
    AND NOT name =~ '.*gds\\\\.(util|similarity|version|alpha\\\\.ml|isLicensed|listProgress|userLog|memory|debug|license|systemMonitor|config|backup|restore).*' 
	AND mode ="READ"
    RETURN name, description, signature, argumentDescription, returnDescription`

	if dbService == nil {
		errMessage := "Database service is not initialized"
		log.Printf("%s", errMessage)
		return mcp.NewToolResultError(errMessage), nil
	}

	// Execute the Cypher query using the database service
	records, err := dbService.ExecuteWriteQuery(ctx, query, nil, config.Database)
	if err != nil {
		// TODO: discuss and write guideline on how to handle tool calling errors.
		log.Printf("Error executing Cypher query: %v", err)
		return mcp.NewToolResultError(err.Error()), nil
	}

	response, err := dbService.Neo4jRecordsToJSON(records)
	if err != nil {
		log.Printf("Error formatting query results: %v", err)
		return mcp.NewToolResultError(err.Error()), nil
	}

	return mcp.NewToolResultText(response), nil
}
