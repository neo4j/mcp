// Copyright (c) "Neo4j"
// Neo4j Sweden AB [http://neo4j.com]

package vector_test

import (
	"strings"
	"testing"

	"github.com/neo4j/mcp/internal/tools/vector"
)

// makeIdx is a test helper that builds a ResolvedIndex from simple fields.
func makeIdx(name, entityType, label, embProp string) *vector.ResolvedIndex {
	return &vector.ResolvedIndex{
		Name:              name,
		EntityType:        entityType,
		Label:             label,
		EmbeddingProperty: embProp,
	}
}

func TestBuildVectorQuery_SearchClause_Node(t *testing.T) {
	t.Parallel()
	idx := makeIdx("moviePlots", "NODE", "Movie", "embedding")
	q, err := vector.BuildVectorQuery(idx, true, true, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertContains(t, q, "ai.text.embed($query, $provider, $embedConfig) AS qv")
	assertContains(t, q, "MATCH (node:`Movie`)")
	assertContains(t, q, "SEARCH node IN (VECTOR INDEX `moviePlots` FOR qv LIMIT $fetchK) SCORE AS score")
	assertContains(t, q, "RETURN node { .*, `embedding`: null } AS node, score")
	assertContains(t, q, "ORDER BY score DESC")
	assertContains(t, q, "LIMIT $topK")
	assertNotContains(t, q, "WHERE")
}

func TestBuildVectorQuery_SearchClause_Node_WithFilters(t *testing.T) {
	t.Parallel()
	idx := makeIdx("moviePlots", "NODE", "Movie", "embedding")
	q, err := vector.BuildVectorQuery(idx, true, true, "node.`genre` = $f0")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertContains(t, q, "WHERE node.`genre` = $f0")
}

func TestBuildVectorQuery_SearchClause_Relationship(t *testing.T) {
	t.Parallel()
	idx := makeIdx("actedInIdx", "RELATIONSHIP", "ACTED_IN", "plotEmbedding")
	q, err := vector.BuildVectorQuery(idx, true, true, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertContains(t, q, "MATCH ()-[node:`ACTED_IN`]->()")
	assertContains(t, q, "VECTOR INDEX `actedInIdx`")
	assertContains(t, q, "`plotEmbedding`: null")
}

func TestBuildVectorQuery_QueryNodes_Node(t *testing.T) {
	t.Parallel()
	idx := makeIdx("moviePlots", "NODE", "Movie", "embedding")
	q, err := vector.BuildVectorQuery(idx, false, true, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertContains(t, q, "CALL db.index.vector.queryNodes($indexName, $fetchK, qv) YIELD node, score")
	assertContains(t, q, "RETURN node { .*, `embedding`: null } AS node, score")
	assertNotContains(t, q, "SEARCH node IN")
	assertNotContains(t, q, "WHERE")
}

func TestBuildVectorQuery_QueryNodes_Node_WithFilters(t *testing.T) {
	t.Parallel()
	idx := makeIdx("moviePlots", "NODE", "Movie", "embedding")
	q, err := vector.BuildVectorQuery(idx, false, true, "node.`year` >= $f0")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertContains(t, q, "WHERE node.`year` >= $f0")
}

func TestBuildVectorQuery_QueryNodes_Relationship(t *testing.T) {
	t.Parallel()
	idx := makeIdx("actedInIdx", "RELATIONSHIP", "ACTED_IN", "plotEmbedding")
	q, err := vector.BuildVectorQuery(idx, false, true, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertContains(t, q, "CALL db.index.vector.queryRelationships($indexName, $fetchK, qv) YIELD relationship AS node, score")
}

func TestBuildVectorQuery_BacktickEscaping(t *testing.T) {
	t.Parallel()
	// Index name and label containing backticks should be doubled
	idx := makeIdx("my`index", "NODE", "My`Label", "emb`Prop")
	q, err := vector.BuildVectorQuery(idx, true, true, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertContains(t, q, "`my``index`")
	assertContains(t, q, "`My``Label`")
	assertContains(t, q, "`emb``Prop`: null")
}

func TestBuildVectorQuery_UseAiTextEmbed_SearchClause(t *testing.T) {
	t.Parallel()
	idx := makeIdx("moviePlots", "NODE", "Movie", "embedding")

	qTrue, err := vector.BuildVectorQuery(idx, true, true, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertContains(t, qTrue, "ai.text.embed($query, $provider, $embedConfig) AS qv")
	assertNotContains(t, qTrue, "genai.vector.encode")

	qFalse, err := vector.BuildVectorQuery(idx, true, false, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertContains(t, qFalse, "genai.vector.encode($query, $provider, $embedConfig) AS qv")
	assertNotContains(t, qFalse, "ai.text.embed")
}

func TestBuildVectorQuery_UseAiTextEmbed_QueryNodes(t *testing.T) {
	t.Parallel()
	idx := makeIdx("moviePlots", "NODE", "Movie", "embedding")

	qTrue, err := vector.BuildVectorQuery(idx, false, true, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertContains(t, qTrue, "ai.text.embed($query, $provider, $embedConfig) AS qv")
	assertNotContains(t, qTrue, "genai.vector.encode")

	qFalse, err := vector.BuildVectorQuery(idx, false, false, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertContains(t, qFalse, "genai.vector.encode($query, $provider, $embedConfig) AS qv")
	assertNotContains(t, qFalse, "ai.text.embed")
}

func TestBuildFilterClause_Empty(t *testing.T) {
	t.Parallel()
	clause, params, err := vector.BuildFilterClause(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if clause != "" {
		t.Errorf("expected empty clause, got %q", clause)
	}
	if len(params) != 0 {
		t.Errorf("expected empty params, got %v", params)
	}
}

func TestBuildFilterClause_SingleEquality(t *testing.T) {
	t.Parallel()
	filters := []vector.Filter{
		{Property: "genre", Operator: "=", Value: "Horror"},
	}
	clause, params, err := vector.BuildFilterClause(filters)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if clause != "node.`genre` = $f0" {
		t.Errorf("unexpected clause: %q", clause)
	}
	if params["f0"] != "Horror" {
		t.Errorf("unexpected param: %v", params["f0"])
	}
}

func TestBuildFilterClause_MultipleFilters(t *testing.T) {
	t.Parallel()
	filters := []vector.Filter{
		{Property: "genre", Operator: "=", Value: "Horror"},
		{Property: "year", Operator: ">=", Value: 2000},
	}
	clause, params, err := vector.BuildFilterClause(filters)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertContains(t, clause, "node.`genre` = $f0")
	assertContains(t, clause, "node.`year` >= $f1")
	assertContains(t, clause, " AND ")
	if params["f0"] != "Horror" {
		t.Errorf("f0 param mismatch: %v", params["f0"])
	}
	if params["f1"] != 2000 {
		t.Errorf("f1 param mismatch: %v", params["f1"])
	}
}

func TestBuildFilterClause_INOperator(t *testing.T) {
	t.Parallel()
	filters := []vector.Filter{
		{Property: "genre", Operator: "IN", Value: []any{"Horror", "SciFi"}},
	}
	clause, params, err := vector.BuildFilterClause(filters)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertContains(t, clause, "node.`genre` IN $f0")
	_ = params
}

func TestBuildFilterClause_INRequiresSlice(t *testing.T) {
	t.Parallel()
	filters := []vector.Filter{
		{Property: "genre", Operator: "IN", Value: "Horror"},
	}
	_, _, err := vector.BuildFilterClause(filters)
	if err == nil {
		t.Error("expected error for IN with non-slice value")
	}
}

func TestBuildFilterClause_AllOperators(t *testing.T) {
	t.Parallel()
	ops := []struct {
		op  string
		val any
	}{
		{"=", "x"},
		{"<>", "x"},
		{"<", 1},
		{"<=", 1},
		{">", 1},
		{">=", 1},
		{"CONTAINS", "x"},
		{"STARTS WITH", "x"},
		{"ENDS WITH", "x"},
		{"IN", []any{"x"}},
	}
	for _, tc := range ops {
		tc := tc
		t.Run(tc.op, func(t *testing.T) {
			t.Parallel()
			filters := []vector.Filter{{Property: "name", Operator: tc.op, Value: tc.val}}
			_, _, err := vector.BuildFilterClause(filters)
			if err != nil {
				t.Errorf("operator %q should be allowed, got error: %v", tc.op, err)
			}
		})
	}
}

func TestBuildFilterClause_OperatorCaseInsensitive(t *testing.T) {
	t.Parallel()
	filters := []vector.Filter{
		{Property: "title", Operator: "contains", Value: "neo"},
	}
	_, _, err := vector.BuildFilterClause(filters)
	if err != nil {
		t.Errorf("lower-case operator should be accepted: %v", err)
	}
}

func TestBuildFilterClause_InvalidOperator(t *testing.T) {
	t.Parallel()
	filters := []vector.Filter{
		{Property: "genre", Operator: "LIKE", Value: "%Horror%"},
	}
	_, _, err := vector.BuildFilterClause(filters)
	if err == nil {
		t.Error("expected error for invalid operator")
	}
}

func TestBuildFilterClause_InjectionAttempts(t *testing.T) {
	t.Parallel()
	badProperties := []string{
		"genre); DROP TABLE Movie; --",
		"a b",
		"1invalid",
		"prop`name",
		"",
		"prop(name)",
	}
	for _, prop := range badProperties {
		prop := prop
		t.Run("bad_prop_"+prop, func(t *testing.T) {
			t.Parallel()
			filters := []vector.Filter{{Property: prop, Operator: "=", Value: "x"}}
			_, _, err := vector.BuildFilterClause(filters)
			if err == nil {
				t.Errorf("expected rejection of property %q", prop)
			}
		})
	}
}

func TestBuildFilterClause_ValidPropertyNames(t *testing.T) {
	t.Parallel()
	goodProperties := []string{"genre", "_internal", "myProp123", "A", "_"}
	for _, prop := range goodProperties {
		prop := prop
		t.Run("good_prop_"+prop, func(t *testing.T) {
			t.Parallel()
			filters := []vector.Filter{{Property: prop, Operator: "=", Value: "x"}}
			_, _, err := vector.BuildFilterClause(filters)
			if err != nil {
				t.Errorf("property %q should be accepted, got: %v", prop, err)
			}
		})
	}
}

// assertContains checks that haystack contains needle.
func assertContains(t *testing.T, haystack, needle string) {
	t.Helper()
	if !strings.Contains(haystack, needle) {
		t.Errorf("expected to contain %q\ngot:\n%s", needle, haystack)
	}
}

// assertNotContains checks that haystack does not contain needle.
func assertNotContains(t *testing.T, haystack, needle string) {
	t.Helper()
	if strings.Contains(haystack, needle) {
		t.Errorf("expected NOT to contain %q\ngot:\n%s", needle, haystack)
	}
}
