// Copyright (c) "Neo4j"
// Neo4j Sweden AB [http://neo4j.com]

package cypher

import (
	"context"
	"log/slog"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/neo4j/mcp/internal/database"
	"github.com/neo4j/mcp/internal/tools"
)

func WriteCypherHandler(deps *tools.ToolDependencies) mcp.ToolHandlerFor[WriteCypherInput, any] {
	return func(ctx context.Context, req *mcp.CallToolRequest, args WriteCypherInput) (*mcp.CallToolResult, any, error) {
		return handleWriteCypher(ctx, args, deps)
	}
}

func handleWriteCypher(ctx context.Context, args WriteCypherInput, deps *tools.ToolDependencies) (*mcp.CallToolResult, any, error) {
	if deps.DBService == nil {
		errMessage := "Database service is not initialized"
		slog.Error(errMessage)
		return tools.NewToolErrorResult(errMessage), nil, nil
	}

	Query := args.Query
	Params := args.Params

	// Validate that query is not empty
	if Query == "" {
		errMessage := "Query parameter is required and cannot be empty"
		slog.Error(errMessage)
		return tools.NewToolErrorResult(errMessage), nil, nil
	}

	// Execute the Cypher query using the database service
	records, err := deps.DBService.ExecuteWriteQuery(ctx, Query, Params)
	if err != nil {
		slog.Error("error executing cypher query", database.ErrorLogAttrs(err)...)
		return tools.NewToolErrorResult(err.Error()), nil, nil
	}

	response, err := deps.DBService.Neo4jRecordsToJSON(records)
	if err != nil {
		slog.Error("error formatting query results", "error", err)
		return tools.NewToolErrorResult(err.Error()), nil, nil
	}

	return tools.NewToolTextResult(response), nil, nil
}
