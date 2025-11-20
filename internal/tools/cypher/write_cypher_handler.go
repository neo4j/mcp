package cypher

import (
	"context"
	"log/slog"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/neo4j/mcp/internal/tools"
)

func WriteCypherHandler(deps *tools.ToolDependencies) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleWriteCypher(ctx, request, deps)
	}
}

func handleWriteCypher(ctx context.Context, request mcp.CallToolRequest, deps *tools.ToolDependencies) (*mcp.CallToolResult, error) {
	if deps.DBService == nil {
		errMessage := "Database service is not initialized"
		slog.Error(errMessage)
		return mcp.NewToolResultError(errMessage), nil
	}

	// Emit analytics event
	if deps.AnalyticsService != nil {
		deps.AnalyticsService.EmitEvent(deps.AnalyticsService.NewToolsEvent("write-cypher"))
	}

	var args WriteCypherInput
	// Use our custom BindArguments that preserves integer types
	if err := BindArguments(request, &args); err != nil {
		slog.Error("error binding arguments", "error", err)
		return mcp.NewToolResultError(err.Error()), nil
	}

	Query := args.Query
	Params := args.Params

	// Validate that query is not empty
	if Query == "" {
		errMessage := "Query parameter is required and cannot be empty"
		slog.Error(errMessage)
		return mcp.NewToolResultError(errMessage), nil
	}

	slog.Info("executing write cypher query", "query", Query)

	lowerCaseQuery := strings.ToLower(Query)
	if strings.Contains(lowerCaseQuery, "call gds.graph.project") {
		if deps.AnalyticsService != nil {
			deps.AnalyticsService.EmitEvent(deps.AnalyticsService.NewGDSProjCreatedEvent())
		}
	}

	if strings.Contains(lowerCaseQuery, "call gds.graph.drop") {
		if deps.AnalyticsService != nil {
			deps.AnalyticsService.EmitEvent(deps.AnalyticsService.NewGDSProjDropEvent())
		}
	}

	// Execute the Cypher query using the database service
	records, err := deps.DBService.ExecuteWriteQuery(ctx, Query, Params)
	if err != nil {
		slog.Error("error executing cypher query", "error", err)
		return mcp.NewToolResultError(err.Error()), nil
	}

	response, err := deps.DBService.Neo4jRecordsToJSON(records)
	if err != nil {
		slog.Error("error formatting query results", "error", err)
		return mcp.NewToolResultError(err.Error()), nil
	}

	return mcp.NewToolResultText(response), nil
}
