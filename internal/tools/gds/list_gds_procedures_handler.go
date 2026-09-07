// Copyright (c) "Neo4j"
// Neo4j Sweden AB [http://neo4j.com]

package gds

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/neo4j/mcp/internal/database"
	"github.com/neo4j/mcp/internal/tools"
)

const listGdsProceduresQuery = `
CALL gds.list() YIELD name, description, signature, type
WHERE type = "procedure"
AND name CONTAINS "stream"
AND NOT (name CONTAINS "estimate")
RETURN name, description, signature, type`

func ListGdsProceduresHandler(deps *tools.ToolDependencies) mcp.ToolHandlerFor[struct{}, any] {
	return func(ctx context.Context, req *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, any, error) {
		return handleListGdsProcedures(ctx, deps)
	}
}

func handleListGdsProcedures(ctx context.Context, deps *tools.ToolDependencies) (*mcp.CallToolResult, any, error) {
	if deps.DBService == nil {
		errMessage := "Database service is not initialized"
		slog.Error(errMessage)
		return tools.NewToolErrorResult(errMessage), nil, nil
	}

	records, err := deps.DBService.ExecuteReadQuery(ctx, listGdsProceduresQuery, nil)
	if err != nil {
		formattedErrorMessage := fmt.Errorf("failed to execute list-gds-procedure query: %v. Ensure that the Graph Data Science (GDS) library is installed and properly configured in your Neo4j database", err)
		slog.Error("failed to execute list gds procedures query", database.ErrorLogAttrs(err)...)
		return tools.NewToolErrorResult(formattedErrorMessage.Error()), nil, nil
	}

	response, err := deps.DBService.Neo4jRecordsToJSON(records)
	if err != nil {
		slog.Error("failed to format list-gds-procedures results to JSON", "error", err)
		return tools.NewToolErrorResult(err.Error()), nil, nil
	}

	return tools.NewToolTextResult(response), nil, nil
}
