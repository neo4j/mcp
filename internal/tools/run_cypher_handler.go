package tools

import (
	"context"
	"fmt"
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
		return mcp.NewToolResultError(err.Error()), nil
	}
	Query := args.Query
	Params := args.Params
	// TODO: handle better these logs during productization process.
	log.Printf("cypher-query: %s", Query)
	if Params != nil {
		log.Printf("cypher-parameters: %v", Params)
	}

	if dbService == nil {
		return mcp.NewToolResultError("Database service is not initialized"), nil
	}

	// Execute the Cypher query using the database service
	records, err := dbService.ExecuteWriteQuery(ctx, Query, Params, config.Database)
	if err != nil {
		// TODO: discuss and write guideline on how to handle tool calling errors.
		return mcp.NewToolResultError(fmt.Sprintf("Error executing Cypher query: %v", err)), nil
	}

	response, err := dbService.Neo4jRecordsToJSON(records)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Error formatting query results: %v", err)), nil
	}

	return mcp.NewToolResultText(response), nil
}
