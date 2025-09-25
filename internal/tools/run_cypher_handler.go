package tools

import (
	"context"
	"fmt"
	"log"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/neo4j/mcp/internal/config"
	"github.com/neo4j/mcp/internal/database"
)

func RunCypherHandler(deps *ToolDependencies) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleRunCypher(ctx, request, deps.DBService, deps.Config, deps.MCPServer)
	}
}

func handleRunCypher(ctx context.Context, request mcp.CallToolRequest, dbService database.DatabaseService, config *config.Config, MCPServer *server.MCPServer) (*mcp.CallToolResult, error) {
	var args RunCypherInput
	// Bind arguments to the struct
	if err := request.BindArguments(&args); err != nil {
		log.Printf("Error binding arguments: %v", err)
		return mcp.NewToolResultError(err.Error()), nil
	}
	Query := args.Query
	Params := args.Params
	// debug log -- to be removed at a later stage
	log.Printf("cypher-query: %s", Query)
	if Params != nil {
		log.Printf("cypher-parameters: %v", Params)
	}

	// Validate that query is not empty
	if Query == "" {
		errMessage := "Query parameter is required and cannot be empty"
		log.Printf("%s", errMessage)
		return mcp.NewToolResultError(errMessage), nil
	}

	if dbService == nil {
		errMessage := "Database service is not initialized"
		log.Printf("%s", errMessage)
		return mcp.NewToolResultError(errMessage), nil
	}
	confirmation, err := requestConfirmation(ctx, Query, MCPServer)

	if confirmation == false {
		// stop cypher execution
		return mcp.NewToolResultError("user has not confirmed the Cypher"), nil
	}

	// Execute the Cypher query using the database service
	records, err := dbService.ExecuteWriteQuery(ctx, Query, Params, config.Database)
	if err != nil {
		// TODO: discuss and write guideline on how to handle tool calling errors.
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

func requestConfirmation(ctx context.Context, Query string, MCPServer *server.MCPServer) (bool, error) {
	description := fmt.Sprintf("Accept risk of executing the following Cypher: %s, by typing: Yes", Query)
	elicitationRequest := mcp.ElicitationRequest{
		Params: mcp.ElicitationParams{
			Message: "Please confirm that you're accepting the risk of executing the following cypher",
			RequestedSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"accept-risk": map[string]any{
						"type":        "string",
						"description": description,
					},
				},
				"required": []string{"accept-risk"},
			},
		},
	}

	// Request elicitation from the client
	result, err := MCPServer.RequestElicitation(ctx, elicitationRequest)
	if err != nil {
		return false, fmt.Errorf("failed to request elicitation: %w", err)
	}

	// Handle the user's response
	switch result.Action {
	case mcp.ElicitationResponseActionAccept:
		log.Printf("User confirmed")
		// do validation logic.
		return true, nil

	case mcp.ElicitationResponseActionDecline:
		// do not execute Cypher
		return false, nil

	case mcp.ElicitationResponseActionCancel:
		// do not execute Cypher
		return false, fmt.Errorf("User cancel elicitation")

	default:
		return false, fmt.Errorf("unexpected response action: %s", result.Action)
	}

}
