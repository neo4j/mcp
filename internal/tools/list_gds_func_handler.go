package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/neo4j/mcp/internal/config"
	"github.com/neo4j/mcp/internal/database"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

// GDSFunction represents a clean, LLM-friendly structure for GDS functions
type GDSFunction struct {
	Name        string              `json:"name"`
	Description string              `json:"description"`
	Signature   string              `json:"signature"`
	Arguments   []FunctionParameter `json:"arguments"`
	Returns     []FunctionParameter `json:"returns"`
}

// FunctionParameter represents a parameter or return value
type FunctionParameter struct {
	Name         string `json:"name"`
	Type         string `json:"type"`
	Description  string `json:"description"`
	DefaultValue string `json:"default,omitempty"`
	IsDeprecated bool   `json:"isDeprecated,omitempty"`
}

// MapGDSFunctionList converts raw Neo4j records to clean GDSFunction structs
func MapGDSFunctionList(records []*neo4j.Record) ([]GDSFunction, error) {
	functions := make([]GDSFunction, 0, len(records))

	for _, record := range records {
		function, err := mapSingleGDSFunction(record)
		if err != nil {
			return nil, fmt.Errorf("failed to map function: %w", err)
		}
		functions = append(functions, function)
	}

	return functions, nil
}

// mapSingleGDSFunction extracts a clean GDSFunction from a Neo4j record
func mapSingleGDSFunction(record *neo4j.Record) (GDSFunction, error) {
	name, _ := record.Get("name")
	description, _ := record.Get("description")
	signature, _ := record.Get("signature")
	argDesc, _ := record.Get("argumentDescription")
	retDesc, _ := record.Get("returnDescription")

	function := GDSFunction{
		Name:        getString(name),
		Description: getString(description),
		Signature:   getString(signature),
		Arguments:   mapParameters(argDesc),
		Returns:     mapParameters(retDesc),
	}

	return function, nil
}

// mapParameters converts Neo4j parameter descriptions to FunctionParameter structs
func mapParameters(params interface{}) []FunctionParameter {
	paramList, ok := params.([]interface{})
	if !ok {
		return []FunctionParameter{}
	}

	result := make([]FunctionParameter, 0, len(paramList))

	for _, p := range paramList {
		paramMap, ok := p.(map[string]interface{})
		if !ok {
			continue
		}

		param := FunctionParameter{
			Name:         getString(paramMap["name"]),
			Type:         getString(paramMap["type"]),
			Description:  getString(paramMap["description"]),
			DefaultValue: getString(paramMap["default"]),
			IsDeprecated: getBool(paramMap["isDeprecated"]),
		}

		result = append(result, param)
	}

	return result
}

// Helper functions for type conversion

func getString(val interface{}) string {
	if val == nil {
		return ""
	}
	if s, ok := val.(string); ok {
		return s
	}
	return fmt.Sprintf("%v", val)
}

func getInt(val interface{}) int {
	if val == nil {
		return 0
	}
	switch v := val.(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	}
	return 0
}

func getFloat(val interface{}) float64 {
	if val == nil {
		return 0
	}
	switch v := val.(type) {
	case float64:
		return v
	case int:
		return float64(v)
	case int64:
		return float64(v)
	}
	return 0
}

func getBool(val interface{}) bool {
	if val == nil {
		return false
	}
	if b, ok := val.(bool); ok {
		return b
	}
	return false
}

func getStringSlice(val interface{}) []string {
	if val == nil {
		return []string{}
	}

	slice, ok := val.([]interface{})
	if !ok {
		return []string{}
	}

	result := make([]string, 0, len(slice))
	for _, item := range slice {
		result = append(result, getString(item))
	}

	return result
}

// ToJSON converts any struct to pretty JSON for LLM consumption
func ToJSON(data interface{}) (string, error) {
	bytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal JSON: %w", err)
	}
	return string(bytes), nil
}

func ListOfGDSFunctionsHandler(deps *ToolDependencies) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleReturnListOfGDSFunctions(ctx, request, deps.DBService, deps.Config)
	}
}

func handleReturnListOfGDSFunctions(ctx context.Context, request mcp.CallToolRequest, dbService database.Service, config *config.Config) (*mcp.CallToolResult, error) {

	// Gets the list of available GDS functions
	query := `SHOW PROCEDURES YIELD * 
	WHERE name STARTS WITH 'gds.' 
	AND name ENDS WITH '.stream'
	AND NOT name CONTAINS '.estimate'
    AND NOT name =~ '.*gds\\\\.(util|similarity|version|alpha\\\\.ml|isLicensed|listProgress|userLog|memory|debug|license|systemMonitor|config|backup|restore).*' 
	AND mode ="READ"
    RETURN name, description, signature, argumentDescription, returnDescription`

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

	//Convert to LLM-friendly JSON
	jsonStr, err := ToJSON(functions)
	if err != nil {
		log.Fatal(err)
	}

	/*
		response, err := dbService.Neo4jRecordsToJSON(records)
		if err != nil {
			log.Printf("Error formatting query results: %v", err)
			return mcp.NewToolResultError(err.Error()), nil
		}
	*/

	return mcp.NewToolResultText(jsonStr), nil
}
