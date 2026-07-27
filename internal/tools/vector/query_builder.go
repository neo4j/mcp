// Copyright (c) "Neo4j"
// Neo4j Sweden AB [http://neo4j.com]

package vector

import (
	"fmt"
	"regexp"
	"strings"
)

// allowedOperators is the set of operators permitted in filter clauses.
var allowedOperators = map[string]bool{
	"=":           true,
	"<>":          true,
	"<":           true,
	"<=":          true,
	">":           true,
	">=":          true,
	"IN":          true,
	"CONTAINS":    true,
	"STARTS WITH": true,
	"ENDS WITH":   true,
}

// safePropertyName matches identifiers that are safe to use as property names.
var safePropertyName = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// BuildFilterClause builds a Cypher WHERE fragment from a list of filters.
// The returned fragment does NOT include the "WHERE" keyword — the caller adds it.
// Property names are validated and backtick-escaped. Values are always bound as
// parameters ($f0, $f1, ...) to prevent injection.
// Returns an empty string and nil params when no filters are provided.
func BuildFilterClause(filters []Filter) (string, map[string]any, error) {
	if len(filters) == 0 {
		return "", nil, nil
	}

	params := make(map[string]any, len(filters))
	parts := make([]string, 0, len(filters))

	for i, f := range filters {
		op := strings.ToUpper(strings.TrimSpace(f.Operator))
		// normalise multi-word operators: collapse internal whitespace
		op = strings.Join(strings.Fields(op), " ")

		if !allowedOperators[op] {
			return "", nil, fmt.Errorf("filter operator '%s' is not allowed; permitted operators: =, <>, <, <=, >, >=, IN, CONTAINS, STARTS WITH, ENDS WITH", f.Operator)
		}

		if !safePropertyName.MatchString(f.Property) {
			return "", nil, fmt.Errorf("filter property '%s' is not a valid identifier; must match ^[A-Za-z_][A-Za-z0-9_]*$", f.Property)
		}

		if op == "IN" {
			// Value must be a slice (any slice type)
			if !isSlice(f.Value) {
				return "", nil, fmt.Errorf("filter operator IN requires a list value for property '%s'", f.Property)
			}
		}

		paramName := fmt.Sprintf("f%d", i)
		params[paramName] = f.Value

		escapedProp := "`" + strings.ReplaceAll(f.Property, "`", "``") + "`"
		parts = append(parts, fmt.Sprintf("node.%s %s $%s", escapedProp, op, paramName))
	}

	return strings.Join(parts, " AND "), params, nil
}

// BuildVectorQuery builds the Cypher query for vector search.
//
//   - idx: resolved index metadata
//   - useSearchClause: true → SEARCH clause (Neo4j ≥ 2026.01); false → db.index.vector.queryNodes
//   - useAiTextEmbed: true → embed via ai.text.embed (Neo4j ≥ 2025.11); false → embed via
//     the deprecated-but-present genai.vector.encode fallback (5.x, 2025.01–2025.10, incl.
//     current Aura 5.x).
//   - filterClause: pre-built WHERE fragment from BuildFilterClause (without the "WHERE" keyword);
//     empty string → no WHERE clause added.
//
// The caller is responsible for merging the filter params from BuildFilterClause into
// the overall query params map.
func BuildVectorQuery(idx *ResolvedIndex, useSearchClause bool, useAiTextEmbed bool, filterClause string) (string, error) {
	escapedLabel, err := escapeIdentifier(idx.Label)
	if err != nil {
		return "", fmt.Errorf("invalid label '%s': %w", idx.Label, err)
	}

	escapedEmbProp, err := escapeIdentifier(idx.EmbeddingProperty)
	if err != nil {
		return "", fmt.Errorf("invalid embedding property '%s': %w", idx.EmbeddingProperty, err)
	}

	whereClause := ""
	if filterClause != "" {
		whereClause = "\nWHERE " + filterClause
	}

	embedFn := "genai.vector.encode"
	if useAiTextEmbed {
		embedFn = "ai.text.embed"
	}

	if useSearchClause {
		escapedIndexName, err := escapeIdentifier(idx.Name)
		if err != nil {
			return "", fmt.Errorf("invalid index name '%s': %w", idx.Name, err)
		}

		var matchLine string
		if idx.EntityType == "RELATIONSHIP" {
			matchLine = fmt.Sprintf("MATCH ()-[node:%s]->()", escapedLabel)
		} else {
			matchLine = fmt.Sprintf("MATCH (node:%s)", escapedLabel)
		}

		query := fmt.Sprintf(
			"WITH %s($query, $provider, $embedConfig) AS qv\n"+
				"%s\n"+
				"  SEARCH node IN (VECTOR INDEX %s FOR qv LIMIT $fetchK) SCORE AS score%s\n"+
				"RETURN node { .*, %s: null } AS node, score\n"+
				"ORDER BY score DESC\n"+
				"LIMIT $topK",
			embedFn,
			matchLine,
			escapedIndexName,
			whereClause,
			escapedEmbProp,
		)
		return query, nil
	}

	// queryNodes / queryRelationships fallback path
	var callLine string
	if idx.EntityType == "RELATIONSHIP" {
		callLine = "CALL db.index.vector.queryRelationships($indexName, $fetchK, qv) YIELD relationship AS node, score"
	} else {
		callLine = "CALL db.index.vector.queryNodes($indexName, $fetchK, qv) YIELD node, score"
	}

	query := fmt.Sprintf(
		"WITH %s($query, $provider, $embedConfig) AS qv\n"+
			"%s%s\n"+
			"RETURN node { .*, %s: null } AS node, score\n"+
			"ORDER BY score DESC\n"+
			"LIMIT $topK",
		embedFn,
		callLine,
		whereClause,
		escapedEmbProp,
	)
	return query, nil
}

// isSlice returns true if v is any slice type (reflects on the interface value).
func isSlice(v any) bool {
	if v == nil {
		return false
	}
	switch v.(type) {
	case []any, []string, []int, []int64, []float64, []bool:
		return true
	}
	return false
}
