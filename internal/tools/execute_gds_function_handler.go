package tools

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/neo4j/mcp/internal/config"
	"github.com/neo4j/mcp/internal/database"
)

// GDSExecutionResult represents the result of executing a GDS function
type GDSExecutionResult struct {
	ProjectionName string                   `json:"projection_name"`
	FunctionName   string                   `json:"function_name"`
	FunctionParams map[string]interface{}   `json:"function_params"`
	Results        []map[string]interface{} `json:"results"`
	ExecutionTime  string                   `json:"execution_time"`
}

// ExecuteGDSFunctionHandler returns a handler function for the execute-gds-function tool
func ExecuteGDSFunctionHandler(deps *ToolDependencies) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleExecuteGDSFunction(ctx, request, deps.DBService, deps.Config)
	}
}

// handleExecuteGDSFunction manages the complete GDS workflow
func handleExecuteGDSFunction(ctx context.Context, request mcp.CallToolRequest, dbService database.Service, config *config.Config) (*mcp.CallToolResult, error) {
	var args ExecuteGDSFunctionInput

	// Bind arguments to the struct
	if err := request.BindArguments(&args); err != nil {
		log.Printf("Error binding arguments: %v", err)
		return mcp.NewToolResultError(err.Error()), nil
	}

	projectionNameStr := "nil"
	if args.ProjectionName != nil {
		projectionNameStr = *args.ProjectionName
	}
	log.Printf("execute gds function - args: %s %v %s", args.FunctionName, args.FunctionParams, projectionNameStr)

	if dbService == nil {
		errMessage := "Database service is not initialized"
		log.Printf("%s", errMessage)
		return mcp.NewToolResultError(errMessage), nil
	}

	startTime := time.Now()

	// Check if projection name is provided
	projectionName := args.ProjectionName
	if projectionName == nil || *projectionName == "" {
		log.Printf("No projection name given")
		return mcp.NewToolResultError(fmt.Sprintf("A projection name is required.")), nil
	}

	/*
		if projectionName == nil || *projectionName == "" {
			generatedName := fmt.Sprintf("gds_projection_%d", startTime.Unix())
			projectionName = &generatedName
		}
	*/

	/*
		// Determine the projection Cypher statement
		projectionCypher := args.ProjectionCypher
		if projectionCypher == nil || strings.TrimSpace(*projectionCypher) == "" {
			defaultProjection := generateDefaultProjection(*projectionName)
			projectionCypher = &defaultProjection
		}
	*/

	log.Printf("Executing GDS function '%s' with projection '%s'", args.FunctionName, *projectionName)

	/*
		// Step 1: Create the projection
		log.Printf("Creating GDS projection: %s", *projectionName)
		_, err := dbService.ExecuteWriteQuery(ctx, *projectionCypher, nil, config.Database)
		if err != nil {
			log.Printf("Error creating GDS projection: %v", err)
			return mcp.NewToolResultError(fmt.Sprintf("Failed to create GDS projection: %v", err)), nil
		}
	*/

	// Ensure cleanup happens even if function execution fails
	defer func() {
		log.Printf("Cleaning up GDS projection: %s", *projectionName)
		dropQuery := fmt.Sprintf("CALL gds.graph.drop('%s')", *projectionName)
		if _, dropErr := dbService.ExecuteWriteQuery(ctx, dropQuery, nil, config.Database); dropErr != nil {
			log.Printf("Warning: Failed to drop GDS projection '%s': %v", *projectionName, dropErr)
		}
	}()

	// Step 2: Execute the GDS function
	log.Printf("Executing GDS function: %s", args.FunctionName)
	functionQuery := fmt.Sprintf("CALL %s('%s', $params)", args.FunctionName, *projectionName)
	functionParams := map[string]interface{}{
		"params": args.FunctionParams,
	}

	records, err := dbService.ExecuteWriteQuery(ctx, functionQuery, functionParams, config.Database)
	if err != nil {
		log.Printf("Error executing GDS function: %v", err)
		return mcp.NewToolResultError(fmt.Sprintf("Failed to execute GDS function '%s': %v", args.FunctionName, err)), nil
	}

	// Step 3: Process the results
	results := make([]map[string]interface{}, len(records))
	for i, record := range records {
		results[i] = record.AsMap()
	}

	executionTime := time.Since(startTime)

	// Step 4: Create the response
	gdsResult := GDSExecutionResult{
		ProjectionName: *projectionName,
		FunctionName:   args.FunctionName,
		FunctionParams: args.FunctionParams,
		Results:        results,
		ExecutionTime:  executionTime.String(),
	}

	// Create standardized LLM response
	LLMResponse := CreateLLMResponse(
		fmt.Sprintf("Successfully executed GDS function '%s' with projection '%s' in %s",
			args.FunctionName, *projectionName, executionTime),
		gdsResult,
		[]string{
			"Analyze the GDS function results to understand the algorithm output",
			"Use the results to make data-driven decisions about your graph",
			"Consider running additional GDS functions if needed for comprehensive analysis",
			"The projection has been automatically cleaned up",
		}...,
	)

	// Convert to LLM-friendly JSON
	jsonStr, err := LLMResponse.ToJSON()
	if err != nil {
		log.Printf("Error formatting LLM response: %v", err)
		return mcp.NewToolResultError(err.Error()), nil
	}

	log.Printf("GDS function execution completed successfully in %s", executionTime)
	return mcp.NewToolResultText(jsonStr), nil
}

// generateDefaultProjection creates a default projection that includes all nodes and relationships
func generateDefaultProjection(projectionName string) string {
	return fmt.Sprintf(`
		CALL gds.graph.project(
			'%s',
			{
				// Project all node labels
				*: {
					label: '*',
					properties: '*'
				}
			},
			{
				// Project all relationship types
				*: {
					type: '*',
					properties: '*'
				}
			}
		)
	`, projectionName)
}

// generateSpecificProjection creates a projection with specific node labels and relationship types
func generateSpecificProjection(projectionName string, nodeLabels []string, relationshipTypes []string) string {
	nodeConfig := make([]string, len(nodeLabels))
	for i, label := range nodeLabels {
		nodeConfig[i] = fmt.Sprintf("'%s': { label: '%s', properties: '*' }", label, label)
	}

	relConfig := make([]string, len(relationshipTypes))
	for i, relType := range relationshipTypes {
		relConfig[i] = fmt.Sprintf("'%s': { type: '%s', properties: '*' }", relType, relType)
	}

	return fmt.Sprintf(`
		CALL gds.graph.project(
			'%s',
			{
				%s
			},
			{
				%s
			}
		)
	`, projectionName, strings.Join(nodeConfig, ", "), strings.Join(relConfig, ", "))
}
