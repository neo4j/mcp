package tools

import (
	"context"
	"log"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/neo4j/mcp/internal/config"
	"github.com/neo4j/mcp/internal/database"
)

func RunCypherHandler(deps *ToolDependencies) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleRunCypher(ctx, request, deps.DBService, deps.Config)
	}
}

func handleRunCypher(ctx context.Context, request mcp.CallToolRequest, dbService database.DatabaseService, config *config.Config) (*mcp.CallToolResult, error) {
	var args RunCypherInput
	// Bind arguments to the struct
	if err := request.BindArguments(&args); err != nil {
		log.Printf("Error binding arguments: %v", err)
		return mcp.NewToolResultError(err.Error()), nil
	}
	Query := args.Query
	Params := args.Params
	// debug log -- to be removed at a later stage
	log.Printf("cypher-query: %s", Query)
	if Params != nil {
		log.Printf("cypher-parameters: %v", Params)
	}

	if dbService == nil {
		errMessage := "Database service is not initialized"
		log.Printf("%s", errMessage)
		return mcp.NewToolResultError(errMessage), nil
	}

	// Execute the Cypher query using the database service
	records, err := dbService.ExecuteWriteQuery(ctx, Query, Params, config.Database)
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
