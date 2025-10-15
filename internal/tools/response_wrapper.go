package tools

import (
	"encoding/json"
	"fmt"
)

// LLMResponseWrapper provides a standardized response format for all MCP tools
// This ensures consistent structure across all tools with Summary, Data, and Next_steps fields
type LLMResponseWrapper[T any] struct {
	Summary   string   `json:"summary"`
	Data      T        `json:"data"`
	NextSteps []string `json:"next_steps,omitempty"`
}

// CreateLLMResponse creates a standardized LLM response with the given data
func CreateLLMResponse[T any](summary string, data T, nextSteps ...string) LLMResponseWrapper[T] {
	return LLMResponseWrapper[T]{
		Summary:   summary,
		Data:      data,
		NextSteps: nextSteps,
	}
}

// ToJSON converts the LLMResponseWrapper to pretty JSON for LLM consumption
func (r LLMResponseWrapper[T]) ToJSON() (string, error) {
	bytes, err := json.MarshalIndent(r, "", "")
	if err != nil {
		return "", fmt.Errorf("failed to marshal LLM response to JSON: %w", err)
	}
	return string(bytes), nil
}

// QueryResult represents the raw data returned from Neo4j queries
// This is used to wrap the JSON string returned by dbService.Neo4jRecordsToJSON
type QueryResult struct {
	RawJSON string `json:"raw_json"`
}

// NewQueryResult creates a new QueryResult from a JSON string
func NewQueryResult(jsonStr string) QueryResult {
	return QueryResult{RawJSON: jsonStr}
}

// Common response summaries that can be reused across tools
const (
	SummarySchemaRetrieved    = "Schema information has been successfully retrieved from the Neo4j database."
	SummaryQueryExecuted      = "Cypher query has been successfully executed and results are available."
	SummaryWriteQueryExecuted = "Write operation has been successfully executed in the Neo4j database."
	SummaryGDSFunctionsListed = "List of available Graph Data Science functions has been retrieved."
	SummaryGDSFunctionDetails = "Detailed information about the requested Graph Data Science function has been retrieved."
)

// Common next steps that can be reused across tools
var (
	NextStepsAfterSchema = []string{
		"Examine the schema to understand the available nodes, relationships, and properties",
		"Use read-cypher to query data from the identified entities",
		"Use write-cypher if you need to modify the database structure or data",
	}

	NextStepsAfterReadQuery = []string{
		"Analyze the returned data to understand the results",
		"Refine your query if needed to get more specific results",
		"Use write-cypher if you need to modify the data",
	}

	NextStepsAfterWriteQuery = []string{
		"Verify the changes by running a read-cypher query",
		"Check if any additional data modifications are needed",
		"Consider updating any related data or relationships",
	}

	NextStepsAfterGDSFunctionList = []string{
		"Choose the GDS function to use based on the description",
		"Use get-gds-function-details to get detailed instructions for how to use it",
	}

	NextStepsAfterGDSFunctionDetails = []string{
		"You then use the execute-gds-function tool to execute a GDS function.  You should always try this tool for running GDS functions.",
		"Step 1 - Create a GDS projection with a unique name.  Remember the projection name as you will need this to use execute-gds-function",
		"Step 2-  Call execute-gds-function with the detailed instructions you got from using get-gds-function-details ",
		"Step 3 - Process the results from execute-gds-function.",
		"execute-gds-function will clean up the projection after it has returned the results to you. ",
	}
)
