// Copyright (c) "Neo4j"
// Neo4j Sweden AB [http://neo4j.com]

package gds

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/neo4j/mcp/internal/tools"
)

func CreateGdsProjectionHandler(deps *tools.ToolDependencies) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleCreateGdsProjection(ctx, req, deps)
	}
}

// TODO: handle differently for AGA vs plugin?
func handleCreateGdsProjection(ctx context.Context, request mcp.CallToolRequest, deps *tools.ToolDependencies) (*mcp.CallToolResult, error) {
	if deps.DBService == nil {
		errMessage := "Database service is not initialized"
		slog.Error(errMessage)
		return mcp.NewToolResultError(errMessage), nil
	}

	var args CreateGdsProjectionInput

	if err := request.BindArguments(&args); err != nil {
		slog.Error("error binding arguments", "error", err)
		return mcp.NewToolResultError(err.Error()), nil
	}

	// TODO: use client facing var names for errs
	if args.Name == "" {
		errMessage := "Name parameter is required and cannot be empty"
		slog.Error(errMessage)
		return mcp.NewToolResultError(errMessage), nil
	}

	if args.ProjectionQuery == "" {
		errMessage := "ProjectionQuery parameter is required and cannot be empty"
		slog.Error(errMessage)
		return mcp.NewToolResultError(errMessage), nil
	}

	if args.SourceNodeVariable == "" {
		errMessage := "SourceNodeVariable parameter is required and cannot be empty"
		slog.Error(errMessage)
		return mcp.NewToolResultError(errMessage), nil
	}

	if args.TargetNodeVariable == "" {
		errMessage := "TargetNodeVariable parameter is required and cannot be empty"
		slog.Error(errMessage)
		return mcp.NewToolResultError(errMessage), nil
	}

	// TODO: avoid cypher injection
	query := strings.Join([]string{
		args.ProjectionQuery,
		// TODO: configurable memory/ttl and only set if using sessions not plugin
		// TODO: use seconds for ttl?
		fmt.Sprintf("RETURN gds.graph.project('%s', %s, %s, {}, {memory: '2GB', ttl: duration({hours: 1})}) AS projectionName", args.Name, args.SourceNodeVariable, args.TargetNodeVariable),
	}, " ")

	records, err := deps.DBService.ExecuteReadQuery(ctx, query, nil)
	if err != nil {
		formattedErrorMessage := fmt.Errorf("failed to execute create projection query: %v. Ensure that the projection query is valid and the source/target node variables are present in the projection query", err)
		slog.Error("failed to execute create projection query", "error", err)
		return mcp.NewToolResultError(formattedErrorMessage.Error()), nil
	}

	response, err := deps.DBService.Neo4jRecordsToJSON(records)
	if err != nil {
		slog.Error("failed to format create projection results to JSON", "error", err)
		return mcp.NewToolResultError(err.Error()), nil
	}

	return mcp.NewToolResultText(response), nil
}
