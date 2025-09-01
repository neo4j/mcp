package tools

import (
	"context"
	"fmt"
	"os"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/neo4j/mcp/internal/config"
	"github.com/neo4j/mcp/internal/database"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

func RunCypherHandler(deps *ToolDependencies) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleRunCypher(ctx, request, deps.Driver, deps.Config)
	}
}

func handleRunCypher(ctx context.Context, request mcp.CallToolRequest, driver *neo4j.DriverWithContext, config *config.Config) (*mcp.CallToolResult, error) {
	var args RunCypherInput
	// Bind arguments to the struct
	if err := request.BindArguments(&args); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	Query := args.Query
	Params := args.Params
	// TODO: handle better these logs during productization process.
	fmt.Fprintf(os.Stderr, "cypher-query: %s\n", Query)
	if Params != nil {
		fmt.Fprintf(os.Stderr, "cypher-parameters: %v\n", Params)
	}

	// Execute the Cypher query using the stored driver
	records, err := database.ExecuteWriteQuery(ctx, driver, Query, Params, config.Database)
	if err != nil {
		// TODO: discuss and write guideline on how to handle tool calling errors.
		return mcp.NewToolResultError(fmt.Sprintf("Error executing Cypher query: %v", err)), nil
	}

	response, err := database.Neo4jRecordsToJSON(records)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Error formatting query results: %v", err)), nil
	}

	return mcp.NewToolResultText(response), nil
}
