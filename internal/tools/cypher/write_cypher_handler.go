package cypher

import (
	"context"
	"log"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/neo4j/mcp/internal/analytics"
	"github.com/neo4j/mcp/internal/database"
	"github.com/neo4j/mcp/internal/tools"
)

func WriteCypherHandler(deps *tools.ToolDependencies) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleWriteCypher(ctx, request, deps.DBService, deps.AnalyticsService)
	}
}

func handleWriteCypher(ctx context.Context, request mcp.CallToolRequest, dbService database.Service, asService analytics.Service) (*mcp.CallToolResult, error) {
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

	asService.EmitEvent(asService.NewToolsEvent("write-cypher"))
	var args WriteCypherInput
	// Use our custom BindArguments that preserves integer types
	if err := BindArguments(request, &args); err != nil {
		log.Printf("Error binding arguments: %v", err)
		return mcp.NewToolResultError(err.Error()), nil
	}

	Query := args.Query
	Params := args.Params
	// debug log -- to be removed at a later stage
	log.Printf("Cypher-query: %s", Query)

	lowerCaseQuery := strings.ToLower(Query)
	if strings.Contains(lowerCaseQuery, "call gds.graph.project") {
		asService.EmitEvent(asService.NewGDSProjCreatedEvent())
	}

	if strings.Contains(lowerCaseQuery, "call gds.graph.drop") {
		asService.EmitEvent(asService.NewGDSProjDropEvent())
	}

	// Validate that query is not empty
	if Query == "" {
		errMessage := "Query parameter is required and cannot be empty"
		log.Printf("%s", errMessage)
		return mcp.NewToolResultError(errMessage), nil
	}

	// Execute the Cypher query using the database service
	records, err := dbService.ExecuteWriteQuery(ctx, Query, Params)
	if err != nil {
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
