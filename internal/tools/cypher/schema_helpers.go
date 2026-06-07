// Copyright (c) "Neo4j"
// Neo4j Sweden AB [http://neo4j.com]

package cypher

import (
	"encoding/json"

	"github.com/mark3labs/mcp-go/mcp"
)

// rawInputSchema sets a raw JSON Schema as the tool's input schema, clearing
// the default object schema that mcp.NewTool pre-populates. Without clearing
// it, Tool.MarshalJSON fails with "provide either InputSchema or
// RawInputSchema, not both".
func rawInputSchema(schema json.RawMessage) mcp.ToolOption {
	return func(t *mcp.Tool) {
		t.InputSchema = mcp.ToolInputSchema{}
		t.RawInputSchema = schema
	}
}
