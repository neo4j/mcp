package database

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

// utility to convert go-driver Records as JSON string
func Neo4jRecordsToJSON(records []*neo4j.Record) (string, error) {
	var results []map[string]any
	for _, record := range records {
		recordMap := make(map[string]any)
		for i, key := range record.Keys {
			recordMap[key] = record.Values[i]
		}
		results = append(results, recordMap)
	}

	formattedResponse, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error formatting response as JSON: %v\n", err)
		return "", err
	}

	formattedResponseStr := string(formattedResponse)
	fmt.Fprintf(os.Stderr, "The formatted response: %s\n", formattedResponseStr)

	return formattedResponseStr, nil
}
