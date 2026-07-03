// Copyright (c) "Neo4j"
// Neo4j Sweden AB [http://neo4j.com]

package vector

// Filter represents a single metadata filter applied during vector search.
type Filter struct {
	Property string `json:"property" jsonschema:"Node property name to filter on"`
	Operator string `json:"operator" jsonschema:"One of: =, <>, <, <=, >, >=, IN, CONTAINS, STARTS WITH, ENDS WITH"`
	Value    any    `json:"value"    jsonschema:"Value to compare against (scalar; list for IN)"`
}

// ResolvedIndex holds resolved metadata about a vector index obtained from SHOW VECTOR INDEXES.
type ResolvedIndex struct {
	Name              string
	EntityType        string // "NODE" or "RELATIONSHIP"
	Label             string // label (node) or relationship type
	EmbeddingProperty string
	Dimensions        int64 // 0 if unknown
}
