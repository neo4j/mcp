package tools

import (
	"context"
	"log"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/neo4j/mcp/internal/config"
	"github.com/neo4j/mcp/internal/database"
)

func WriteCypherHandler(deps *ToolDependencies) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleWriteCypher(ctx, request, deps.DBService, deps.Config)
	}
}

func handleWriteCypher(ctx context.Context, request mcp.CallToolRequest, dbService database.Service, config *config.Config) (*mcp.CallToolResult, error) {
	var args WriteCypherInput
	// Bind arguments to the struct
	if err := request.BindArguments(&args); err != nil {
		log.Printf("Error binding arguments: %v", err)
		return mcp.NewToolResultError(err.Error()), nil
	}
	Query := args.Query
	Params := args.Params
	// debug log -- to be removed at a later stage
	log.Printf("Cypher-query: %s", Query)

	// Validate that query is not empty
	if Query == "" {
		errMessage := "Query parameter is required and cannot be empty"
		log.Printf("%s", errMessage)
		return mcp.NewToolResultError(errMessage), nil
	}

	if dbService == nil {
		errMessage := "Database service is not initialized"
		log.Printf("%s", errMessage)
		return mcp.NewToolResultError(errMessage), nil
	}

	// Execute the Cypher query using the database service
	records, err := dbService.ExecuteWriteQuery(ctx, Query, Params, config.Database)
	if err != nil {
		log.Printf("Error executing Cypher query: %v", err)
		return mcp.NewToolResultError(err.Error()), nil
	}

	response, err := dbService.Neo4jRecordsToJSON(records)
	if err != nil {
		log.Printf("Error formatting query results: %v", err)
		return mcp.NewToolResultError(err.Error()), nil
	}

	//Create standardized LLM response
	LLMResponse := CreateLLMResponse(
		SummaryWriteQueryExecuted,
		NewQueryResult(response),
		NextStepsAfterWriteQuery...,
	)

	//Convert to LLM-friendly JSON
	jsonStr, err := LLMResponse.ToJSON()
	if err != nil {
		log.Printf("Error formatting LLM response: %v", err)
		return mcp.NewToolResultError(err.Error()), nil
	}

	return mcp.NewToolResultText(jsonStr), nil
}
