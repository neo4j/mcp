// Copyright (c) "Neo4j"
// Neo4j Sweden AB [http://neo4j.com]

package cypher

import (
	"context"
	"log/slog"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/neo4j/mcp/internal/database"
	"github.com/neo4j/mcp/internal/tools"
	"github.com/neo4j/neo4j-go-driver/v6/neo4j"
)

func ReadCypherHandler(deps *tools.ToolDependencies) mcp.ToolHandlerFor[ReadCypherInput, any] {
	return func(ctx context.Context, req *mcp.CallToolRequest, args ReadCypherInput) (*mcp.CallToolResult, any, error) {
		return handleReadCypher(ctx, args, deps)
	}
}

func handleReadCypher(ctx context.Context, args ReadCypherInput, deps *tools.ToolDependencies) (*mcp.CallToolResult, any, error) {
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

	// Get queryType by pre-appending "EXPLAIN" to identify if the query is of type "r", if not raise a ToolResultError
	queryType, err := deps.DBService.GetQueryType(ctx, Query, Params)
	if err != nil {
		slog.Error("error classifying cypher query", database.ErrorLogAttrs(err)...)
		return tools.NewToolErrorResult(err.Error()), nil, nil
	}

	if queryType != neo4j.QueryTypeReadOnly { // only queryType == "r" are allowed in read-cypher
		errMessage := "read-cypher can only run read-only Cypher statements. For write operations (CREATE, MERGE, DELETE, SET, etc...), schema/admin commands, or PROFILE queries, use write-cypher instead."
		slog.Error("rejected non-read query", "type", queryType)
		return tools.NewToolErrorResult(errMessage), nil, nil
	}

	// Execute the Cypher query using the database service (now confirmed read-only)
	records, err := deps.DBService.ExecuteReadQuery(ctx, Query, Params)
	if err != nil {
		slog.Error("error executing cypher query", database.ErrorLogAttrs(err)...)
		return tools.NewToolErrorResult(err.Error()), nil, nil
	}

	// Format records to JSON
	response, err := deps.DBService.Neo4jRecordsToJSON(records)
	if err != nil {
		slog.Error("error formatting query results", "error", err)
		return tools.NewToolErrorResult(err.Error()), nil, nil
	}

	return tools.NewToolTextResult(response), nil, nil
}
