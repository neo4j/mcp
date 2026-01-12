package cypher

import (
	"context"
	"log/slog"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/neo4j/mcp/internal/tools"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

func ReadCypherHandler(deps *tools.ToolDependencies) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleReadCypher(ctx, request, deps)
	}
}

func handleReadCypher(ctx context.Context, request mcp.CallToolRequest, deps *tools.ToolDependencies) (*mcp.CallToolResult, error) {
	if deps.DBService == nil {
		errMessage := "Database service is not initialized"
		slog.Error(errMessage)
		return mcp.NewToolResultError(errMessage), nil
	}

	var args ReadCypherInput

	if err := request.BindArguments(&args); err != nil {
		slog.Error("error binding arguments", "error", err)
		return mcp.NewToolResultError(err.Error()), nil
	}
	Query := args.Query
	Params := args.Params

	slog.Info("executing read cypher query", "query", Query)

	// Validate that query is not empty
	if Query == "" {
		errMessage := "Query parameter is required and cannot be empty"
		slog.Error(errMessage)
		return mcp.NewToolResultError(errMessage), nil
	}

	// Get queryType by pre-appending "EXPLAIN" to identify if the query is of type "r", if not raise a ToolResultError
	queryType, err := deps.DBService.GetQueryType(ctx, Query, Params)
	if err != nil {
		slog.Error("error classifying cypher query", "error", err)
		return mcp.NewToolResultError(err.Error()), nil
	}

	if queryType != neo4j.StatementTypeReadOnly { // only queryType == "r" are allowed in read-cypher
		errMessage := "read-cypher can only run read-only Cypher statements. For write operations (CREATE, MERGE, DELETE, SET, etc...), schema/admin commands, or PROFILE queries, use write-cypher instead."
		slog.Error("rejected non-read query", "type", queryType, "query", Query)
		return mcp.NewToolResultError(errMessage), nil
	}

	// Execute the Cypher query using the database service (now confirmed read-only)
	records, err := deps.DBService.ExecuteReadQuery(ctx, Query, Params)
	if err != nil {
		slog.Error("error executing cypher query", "error", err)
		return mcp.NewToolResultError(err.Error()), nil
	}

	// Format records to JSON
	response, err := deps.DBService.Neo4jRecordsToJSON(records)
	if err != nil {
		slog.Error("error formatting query results", "error", err)
		return mcp.NewToolResultError(err.Error()), nil
	}

	return mcp.NewToolResultText(response), nil
}
