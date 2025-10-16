package tools

import (
	"context"
	"fmt"
	"log"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/neo4j/mcp/internal/config"
	"github.com/neo4j/mcp/internal/database"
)

const listGdsProceduresQuery = `
CALL gds.list() YIELD name, description, signature, type
WHERE type = "procedure"
AND name CONTAINS "stream"
AND NOT (name CONTAINS "estimate")
RETURN name, description, signature, type`

func ListGdsProceduresHandler(deps *ToolDependencies) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleListGdsProcedures(ctx, deps.DBService, deps.Config)
	}
}

func handleListGdsProcedures(ctx context.Context, dbService database.Service, config *config.Config) (*mcp.CallToolResult, error) {
	if dbService == nil {
		errMessage := "Database service is not initialized"
		log.Printf("%s", errMessage)
		return mcp.NewToolResultError(errMessage), nil
	}
	records, err := dbService.ExecuteReadQuery(ctx, listGdsProceduresQuery, nil, config.Database)
	if err != nil {
		formattedErrorMessage := fmt.Errorf("failed to execute list-gds-procedure query: %v. Ensure that the Graph Data Science (GDS) library is installed and properly configured in your Neo4j database", err)
		log.Printf("%s", formattedErrorMessage.Error())
		return mcp.NewToolResultError(formattedErrorMessage.Error()), nil
	}

	response, err := dbService.Neo4jRecordsToJSON(records)
	if err != nil {
		log.Printf("Failed to format list-gds-procedures results to JSON: %v", err)
		return mcp.NewToolResultError(err.Error()), nil
	}

	return mcp.NewToolResultText(response), nil
}
