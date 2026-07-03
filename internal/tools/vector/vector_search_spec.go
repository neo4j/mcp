// Copyright (c) "Neo4j"
// Neo4j Sweden AB [http://neo4j.com]

package vector

import (
	"github.com/mark3labs/mcp-go/mcp"
)

// SearchInput defines the arguments accepted by the vector-search tool.
type SearchInput struct {
	Query     string   `json:"query"               jsonschema:"Natural-language text to semantically search for"`
	IndexName string   `json:"indexName,omitempty" jsonschema:"Name of the vector index to search. If omitted and exactly one vector index exists, it is used automatically."`
	TopK      int      `json:"topK,omitempty"      jsonschema:"Number of nearest neighbors to return (default 5, max 100)"`
	Filters   []Filter `json:"filters,omitempty"   jsonschema:"Optional metadata filters applied to results (AND-combined)"`
}

// SearchSpec returns the MCP tool definition for vector-search.
func SearchSpec() mcp.Tool {
	return mcp.NewTool("vector-search",
		mcp.WithDescription(
			"Semantic vector search over a Neo4j vector index. Embeds the query text and "+
				"returns the most similar nodes with similarity scores. Use get-schema to discover "+
				"available vector indexes and their node labels/properties."),
		mcp.WithInputSchema[SearchInput](),
		mcp.WithTitleAnnotation("Vector Search"),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
		mcp.WithOpenWorldHintAnnotation(true),
	)
}
