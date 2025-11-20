package cypher

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/neo4j/mcp/internal/tools"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

const (
	// schemaQuery is the APOC query used to retrieve comprehensive schema information
	schemaQuery = `
        CALL apoc.meta.schema({sample: $sampleSize})
        YIELD value
        UNWIND keys(value) as key
        WITH key, value[key] as value
        RETURN key, value { .properties, .type, .relationships } as value
    `
)

// GetSchemaHandler returns a handler function for the get_schema tool
func GetSchemaHandler(deps *tools.ToolDependencies) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return handleGetSchema(ctx, request, deps)
	}
}

// handleGetSchema retrieves Neo4j schema information using APOC
func handleGetSchema(ctx context.Context, request mcp.CallToolRequest, deps *tools.ToolDependencies) (*mcp.CallToolResult, error) {
	if deps.DBService == nil {
		errMessage := "database service is not initialized"
		slog.Error(errMessage)
		return mcp.NewToolResultError(errMessage), nil
	}
	// Emit analytics event
	if deps.AnalyticsService == nil {
		errMessage := "analytics service is not initialized"
		slog.Error(errMessage)
		return mcp.NewToolResultError(errMessage), nil
	}
	var args GetSchemaInput

	// Use our custom BindArguments that preserves integer types
	if err := request.BindArguments(&args); err != nil {
		slog.Error("error binding arguments", "error", err)
		return mcp.NewToolResultError(err.Error()), nil
	}

	deps.AnalyticsService.EmitEvent(deps.AnalyticsService.NewToolsEvent("get-schema"))
	slog.Info("retrieving schema from the database")

	// Execute the APOC schema query
	records, err := deps.DBService.ExecuteReadQuery(ctx, schemaQuery, map[string]any{
		"sampleSize": args.SampleSize,
	})
	if err != nil {
		slog.Error("failed to execute schema query", "error", err)
		return mcp.NewToolResultError(err.Error()), nil
	}
	if len(records) == 0 {
		slog.Warn("schema is empty, no data in the database")
		return mcp.NewToolResultText("The get-schema tool executed successfully; however, since the Neo4j instance contains no data, no schema information was returned."), nil
	}
	structuredOutput, err := processCypherSchema(records)
	if err != nil {
		slog.Error("failed to process get-schema Cypher Query", "error", err)
		return mcp.NewToolResultError(err.Error()), nil
	}
	jsonData, err := json.Marshal(structuredOutput)
	if err != nil {
		slog.Error("failed to serialize structured schema", "error", err)
		return mcp.NewToolResultError(err.Error()), nil

	}
	return mcp.NewToolResultText(string(jsonData)), nil
}

type SchemaItem struct {
	Key   string       `json:"key"`
	Value SchemaDetail `json:"value"`
}

type SchemaDetail struct {
	Type          string                  `json:"type"`
	Properties    map[string]string       `json:"properties,omitempty"`
	Relationships map[string]Relationship `json:"relationships,omitempty"`
}

type Relationship struct {
	Direction  string            `json:"direction"`
	Labels     []string          `json:"labels"` // List of target node labels
	Properties map[string]string `json:"properties,omitempty"`
}

// processCypherSchema is a func that transform a list of Neo4j.Record in a JSON tagged struct,
// this allow to maintain the same APOC query supported by multiple Neo4j versions while returning a tokens aware version of it.
// Properties are optimized to return directly the type and removing unnecessary information:
// From:
//
//	title: {
//	     unique: false,
//	     indexed: false,
//	     type: "STRING",
//	     existence: false
//	   }
//
// To:
// title: String
// Relationship,
// From:
//
//	 relationships:   {
//	    ALWAYS: {
//	      count: 16,
//	      direction: "out",
//	      labels: ["Something"],
//	      properties: {
//				releaseYear: {
//	      		unique: false,
//	      		indexed: false,
//	      		type: "STRING",
//	      		existence: false
//	    		}
//			 }
//	    }
//	  }
//
// To:
// { ALWAYS: { direction: "out", labels: ["ACTED_IN"], properties: { releaseYear: "DATE" } } }
// null values are stripped.
func processCypherSchema(records []*neo4j.Record) ([]SchemaItem, error) {
	simplifiedSchema := make([]SchemaItem, 0, len(records))

	for _, record := range records {
		// Extract "key" (e.g., "Movie", "ACTED_IN")
		keyRaw, ok := record.Get("key")
		if !ok {
			return nil, fmt.Errorf("missing 'key' column in record")
		}
		keyStr, _ := keyRaw.(string)

		// Extract "value" (The map containing properties, type, relationships)
		valRaw, ok := record.Get("value")
		if !ok {
			return nil, fmt.Errorf("missing 'value' column in record")
		}

		// Cast the value to a map
		data, ok := valRaw.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("invalid value returned")
		}

		// Transformation logic.

		//  Extract Type ("node" or "relationship")
		itemType, _ := data["type"].(string)

		// Simplify Properties
		// Input:  { "name": { "type": "STRING", "indexed": ... } }
		// Output: { "name": "STRING" }
		cleanProps := simplifyProperties(data["properties"])

		// Simplify Relationships
		// Input:  { "CONNECTION": { "relationship": null, "direction": "out", "properties": {...} } }
		// Output: { "CONNECTION": { "direction": "out", "properties": {"dist": "FLOAT"} } }
		var cleanRels map[string]Relationship

		if rawRels, ok := data["relationships"].(map[string]interface{}); ok && len(rawRels) > 0 {
			cleanRels = make(map[string]Relationship)
			for relName, rawRelDetails := range rawRels {
				if relDetails, ok := rawRelDetails.(map[string]interface{}); ok {

					// Extract Direction
					direction, _ := relDetails["direction"].(string)

					// Extract Target Labels
					var labels []string
					if rawLabels, ok := relDetails["labels"].([]interface{}); ok {
						for _, l := range rawLabels {
							if lStr, ok := l.(string); ok {
								labels = append(labels, lStr)
							}
						}
					}

					relProps := simplifyProperties(relDetails["properties"])

					cleanRels[relName] = Relationship{
						Direction:  direction,
						Labels:     labels,
						Properties: relProps,
					}
				}
			}
		}

		simplifiedSchema = append(simplifiedSchema, SchemaItem{
			Key: keyStr,
			Value: SchemaDetail{
				Type:          itemType,
				Properties:    cleanProps,
				Relationships: cleanRels,
			},
		})
	}

	return simplifiedSchema, nil
}

func simplifyProperties(rawProps interface{}) map[string]string {
	cleanProps := make(map[string]string)
	if props, ok := rawProps.(map[string]interface{}); ok {
		for propName, rawPropDetails := range props {
			if propDetails, ok := rawPropDetails.(map[string]interface{}); ok {
				if typeName, ok := propDetails["type"].(string); ok {
					cleanProps[propName] = typeName
				}
			}
		}
	}
	return cleanProps
}
