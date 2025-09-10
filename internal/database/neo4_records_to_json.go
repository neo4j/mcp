package database

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

// utility to convert go-driver Records as JSON string
func Neo4jRecordsToJSON(records []*neo4j.Record) (string, error) {
	var results []map[string]any
	for _, record := range records {
		recordMap := record.AsMap()
		results = append(results, recordMap)
	}

	formattedResponse, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to format records as JSON: %w", err)
	}

	formattedResponseStr := string(formattedResponse)
	log.Printf("The formatted response: %s", formattedResponseStr)

	return formattedResponseStr, nil
}
