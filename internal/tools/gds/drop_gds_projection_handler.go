// Copyright (c) "Neo4j"
// Neo4j Sweden AB [http://neo4j.com]

package gds

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/neo4j/mcp/internal/tools"
)

const dropGdsProjectionQueryFmt = "CALL gds.graph.drop('%s') YIELD graphName;"

func DropGdsProjectionHandler(deps *tools.ToolDependencies) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleDropGdsProjection(ctx, req, deps)
	}
}

func handleDropGdsProjection(ctx context.Context, request mcp.CallToolRequest, deps *tools.ToolDependencies) (*mcp.CallToolResult, error) {
	if deps.DBService == nil {
		errMessage := "Database service is not initialized"
		slog.Error(errMessage)
		return mcp.NewToolResultError(errMessage), nil
	}

	var args DropGdsProjectionInput
	if err := request.BindArguments(&args); err != nil {
		slog.Error("error binding arguments", "error", err)
		return mcp.NewToolResultError(err.Error()), nil
	}

	// TODO: use client facing var names for errs
	if args.ProjectionName == "" {
		errMessage := "ProjectionName parameter is required and cannot be empty"
		slog.Error(errMessage)
		return mcp.NewToolResultError(errMessage), nil
	}

	// TODO: avoid cypher injection
	records, err := deps.DBService.ExecuteReadQuery(ctx, fmt.Sprintf(dropGdsProjectionQueryFmt, args.ProjectionName), nil)
	if err != nil {
		formattedErrorMessage := fmt.Errorf("failed to execute list-gds-procedure query: %v. Ensure that the Graph Data Science (GDS) library is installed and properly configured in your Neo4j database", err)
		slog.Error("failed to execute list gds procedures query", "error", err)
		return mcp.NewToolResultError(formattedErrorMessage.Error()), nil
	}

	response, err := deps.DBService.Neo4jRecordsToJSON(records)
	if err != nil {
		slog.Error("failed to format list-gds-procedures results to JSON", "error", err)
		return mcp.NewToolResultError(err.Error()), nil
	}

	return mcp.NewToolResultText(response), nil
}
