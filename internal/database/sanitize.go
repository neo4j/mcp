// Copyright (c) "Neo4j"
// Neo4j Sweden AB [http://neo4j.com]

package database

import (
	"fmt"

	"github.com/neo4j/neo4j-go-driver/v6/neo4j"
)

// embeddingMinLength is the length from which a list of floats or a native VECTOR is treated as an embedding.
// It matches the list limit used by neo4j-graphrag-python and the Labs MCP servers.
const embeddingMinLength = 128

// sanitizeValue returns a copy of v with embedding-like values replaced by a short placeholder.
// Embeddings are informationally worthless to an LLM and can exhaust its context window.
func sanitizeValue(v any) any {
	switch val := v.(type) {
	case map[string]any:
		sanitized := make(map[string]any, len(val))
		for k, item := range val {
			sanitized[k] = sanitizeValue(item)
		}
		return sanitized
	case []any:
		if isEmbedding(val) {
			return embeddingPlaceholder(len(val))
		}
		sanitized := make([]any, len(val))
		for i, item := range val {
			sanitized[i] = sanitizeValue(item)
		}
		return sanitized
	case neo4j.Node:
		return sanitizeNode(val)
	case neo4j.Relationship:
		return sanitizeRelationship(val)
	case neo4j.Path:
		nodes := make([]neo4j.Node, len(val.Nodes))
		for i, n := range val.Nodes {
			nodes[i] = sanitizeNode(n)
		}
		relationships := make([]neo4j.Relationship, len(val.Relationships))
		for i, r := range val.Relationships {
			relationships[i] = sanitizeRelationship(r)
		}
		return neo4j.Path{Nodes: nodes, Relationships: relationships}
	default:
		if length, ok := vectorLength(v); ok && length >= embeddingMinLength {
			return embeddingPlaceholder(length)
		}
		return v
	}
}

// vectorLength returns the length of a native Neo4j VECTOR value, which the driver returns as
// neo4j.Vector[T] rather than as a list.
func vectorLength(v any) (int, bool) {
	switch val := v.(type) {
	case neo4j.Vector[float64]:
		return len(val.Elems), true
	case neo4j.Vector[float32]:
		return len(val.Elems), true
	case neo4j.Vector[int64]:
		return len(val.Elems), true
	case neo4j.Vector[int32]:
		return len(val.Elems), true
	case neo4j.Vector[int16]:
		return len(val.Elems), true
	case neo4j.Vector[int8]:
		return len(val.Elems), true
	default:
		return 0, false
	}
}

// sanitizeNode and sanitizeRelationship return a copy with new Props; the original Props map is
// shared with the caller's record and must not be modified.
func sanitizeNode(n neo4j.Node) neo4j.Node {
	n.Props = sanitizeValue(n.Props).(map[string]any)
	return n
}

func sanitizeRelationship(r neo4j.Relationship) neo4j.Relationship {
	r.Props = sanitizeValue(r.Props).(map[string]any)
	return r
}

// isEmbedding reports whether a list has the shape of a stored embedding: long and all floats.
// Long lists of other values, such as collect(m.title), are results the caller asked for.
func isEmbedding(list []any) bool {
	if len(list) < embeddingMinLength {
		return false
	}
	for _, item := range list {
		if _, ok := item.(float64); !ok {
			return false
		}
	}
	return true
}

func embeddingPlaceholder(length int) string {
	return fmt.Sprintf("<embedding: %d values omitted>", length)
}
