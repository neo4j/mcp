// Copyright (c) "Neo4j"
// Neo4j Sweden AB [http://neo4j.com]

package cypher

import (
	"fmt"

	"github.com/neo4j/neo4j-go-driver/v6/neo4j"
)

// embeddingMinLength is the length from which a list of floats or a native VECTOR is treated as an embedding.
// It matches the list limit used by neo4j-graphrag-python and the Labs MCP servers.
const embeddingMinLength = 128

// omitEmbeddings returns copies of records with embedding-like values replaced by a short placeholder.
// It is applied only when the caller sets omitEmbeddings, so the default result path is unchanged.
// The input records are not modified.
func omitEmbeddings(records []*neo4j.Record) []*neo4j.Record {
	sanitized := make([]*neo4j.Record, len(records))
	for i, record := range records {
		values := make([]any, len(record.Values))
		for j, v := range record.Values {
			values[j] = omitEmbeddingValue(v)
		}
		sanitized[i] = &neo4j.Record{Keys: record.Keys, Values: values}
	}
	return sanitized
}

func omitEmbeddingValue(v any) any {
	switch val := v.(type) {
	case map[string]any:
		if val == nil {
			return val
		}
		sanitized := make(map[string]any, len(val))
		for k, item := range val {
			sanitized[k] = omitEmbeddingValue(item)
		}
		return sanitized
	case []any:
		if isEmbedding(val) {
			return embeddingPlaceholder(len(val))
		}
		sanitized := make([]any, len(val))
		for i, item := range val {
			sanitized[i] = omitEmbeddingValue(item)
		}
		return sanitized
	case neo4j.Node:
		return omitNodeEmbeddings(val)
	case neo4j.Relationship:
		return omitRelationshipEmbeddings(val)
	case neo4j.Path:
		nodes := make([]neo4j.Node, len(val.Nodes))
		for i, n := range val.Nodes {
			nodes[i] = omitNodeEmbeddings(n)
		}
		relationships := make([]neo4j.Relationship, len(val.Relationships))
		for i, r := range val.Relationships {
			relationships[i] = omitRelationshipEmbeddings(r)
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

// omitNodeEmbeddings and omitRelationshipEmbeddings return a copy with new Props; the original
// Props map is shared with the driver's record and must not be modified.
func omitNodeEmbeddings(n neo4j.Node) neo4j.Node {
	n.Props = omitEmbeddingValue(n.Props).(map[string]any)
	return n
}

func omitRelationshipEmbeddings(r neo4j.Relationship) neo4j.Relationship {
	r.Props = omitEmbeddingValue(r.Props).(map[string]any)
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
