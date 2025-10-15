package tools

import (
	"context"
	"fmt"
	"log"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/neo4j/mcp/internal/config"
	"github.com/neo4j/mcp/internal/database"
)

// LLMGDSFunctionResponseWrapper is deprecated - use LLMResponseWrapper[[]GDSFunction] instead
// This type is kept for backward compatibility but new code should use the standardized response format
type LLMGDSFunctionResponseWrapper struct {
	Type         string
	Summary      string
	Data         []GDSFunction
	Next_actions map[string]string
}

func GetGDSFunctionDetailsHandler(deps *ToolDependencies) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleGetGDSFunctionDetails(ctx, request, deps.DBService, deps.Config)
	}
}

func handleGetGDSFunctionDetails(ctx context.Context, request mcp.CallToolRequest, dbService database.Service, config *config.Config) (*mcp.CallToolResult, error) {
	var args GetGDSFunctionDetailsInput
	// Bind arguments to the struct
	if err := request.BindArguments(&args); err != nil {
		log.Printf("Error binding arguments: %v", err)
		return mcp.NewToolResultError(err.Error()), nil
	}

	// Gets the list of available GDS functions
	query := fmt.Sprintf(`SHOW PROCEDURES YIELD * WHERE name = '%s' RETURN name, description, signature, argumentDescription, returnDescription`, args.FunctionName)

	if dbService == nil {
		errMessage := "Database service is not initialized"
		log.Printf("%s", errMessage)
		return mcp.NewToolResultError(errMessage), nil
	}

	// Execute the Cypher query using the database service
	records, err := dbService.ExecuteWriteQuery(ctx, query, nil, config.Database)
	if err != nil {
		// TODO: discuss and write guideline on how to handle tool calling errors.
		log.Printf("Error executing Cypher query: %v", err)
		return mcp.NewToolResultError(err.Error()), nil
	}

	//Convert to clean structs
	functions, err := MapGDSFunctionList(records)
	if err != nil {
		log.Fatal(err)
	}

	//Create standardized LLM response
	LLMResponse := CreateLLMResponse(
		SummaryGDSFunctionDetails,
		functions,
		NextStepsAfterGDSFunctionDetails...,
	)

	//Convert to LLM-friendly JSON
	jsonStr, err := LLMResponse.ToJSON()
	if err != nil {
		log.Fatal(err)
	}

	return mcp.NewToolResultText(jsonStr), nil
}
