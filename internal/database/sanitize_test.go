// Copyright (c) "Neo4j"
// Neo4j Sweden AB [http://neo4j.com]

package database_test

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/neo4j/mcp/internal/database"
	"github.com/neo4j/neo4j-go-driver/v6/neo4j"
)

// floats returns n float values, the shape the driver produces for a LIST<FLOAT>
// property such as a stored embedding.
func floats(n int) []any {
	l := make([]any, n)
	for i := range l {
		l[i] = 0.25
	}
	return l
}

// recordsToMaps runs Neo4jRecordsToJSON and decodes its output, so tests can assert
// on the structure an LLM client receives rather than on whitespace.
func recordsToMaps(t *testing.T, records ...*neo4j.Record) []map[string]any {
	t.Helper()
	var s database.Neo4jService
	out, err := s.Neo4jRecordsToJSON(records)
	if err != nil {
		t.Fatalf("Neo4jRecordsToJSON() error = %v", err)
	}
	var decoded []map[string]any
	if err := json.Unmarshal([]byte(out), &decoded); err != nil {
		t.Fatalf("output is not valid JSON: %v\n%s", err, out)
	}
	return decoded
}

func TestNeo4jRecordsToJSON_OmitsEmbeddingLists(t *testing.T) {
	tests := []struct {
		name   string
		record *neo4j.Record
		want   map[string]any
	}{
		{
			name:   "embedding-sized list of floats is replaced with a placeholder",
			record: newTestRecord([]string{"embedding"}, []any{floats(1536)}),
			want:   map[string]any{"embedding": "<embedding: 1536 values omitted>"},
		},
		{
			name:   "list of 128 floats is replaced",
			record: newTestRecord([]string{"v"}, []any{floats(128)}),
			want:   map[string]any{"v": "<embedding: 128 values omitted>"},
		},
		{
			name:   "list of 127 floats is kept",
			record: newTestRecord([]string{"v"}, []any{floats(127)}),
			want:   map[string]any{"v": floats(127)},
		},
		{
			name: "embedding inside a map is replaced and sibling keys are kept",
			record: newTestRecord([]string{"m"}, []any{
				map[string]any{"title": "Heat", "embedding": floats(1536)},
			}),
			want: map[string]any{"m": map[string]any{
				"title":     "Heat",
				"embedding": "<embedding: 1536 values omitted>",
			}},
		},
		{
			name: "embeddings inside a short list of maps are replaced",
			record: newTestRecord([]string{"rows"}, []any{[]any{
				map[string]any{"embedding": floats(1536)},
				map[string]any{"embedding": floats(768)},
			}}),
			want: map[string]any{"rows": []any{
				map[string]any{"embedding": "<embedding: 1536 values omitted>"},
				map[string]any{"embedding": "<embedding: 768 values omitted>"},
			}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := recordsToMaps(t, tt.record)
			if len(got) != 1 {
				t.Fatalf("got %d rows, want 1", len(got))
			}
			if !reflect.DeepEqual(got[0], tt.want) {
				t.Fatalf("row = %#v\nwant  %#v", got[0], tt.want)
			}
		})
	}
}

// Long lists that are not all floats are query results the caller asked for,
// e.g. collect(m.title) or collect(p.id), and must come back intact.
func TestNeo4jRecordsToJSON_KeepsLongNonFloatLists(t *testing.T) {
	titles := make([]any, 300)
	ids := make([]any, 300)
	mixed := floats(300)
	for i := range titles {
		titles[i] = "Heat"
		ids[i] = int64(42)
	}
	mixed[299] = nil

	got := recordsToMaps(t, newTestRecord(
		[]string{"titles", "ids", "mixed"},
		[]any{titles, ids, mixed},
	))

	for _, key := range []string{"titles", "ids", "mixed"} {
		list, ok := got[0][key].([]any)
		if !ok {
			t.Fatalf("%s = %#v, want the original list", key, got[0][key])
		}
		if len(list) != 300 {
			t.Fatalf("%s has %d elements, want 300", key, len(list))
		}
	}
}

func TestNeo4jRecordsToJSON_OmitsEmbeddingsFromGraphEntities(t *testing.T) {
	movie := neo4j.Node{
		ElementId: "4:db:1",
		Labels:    []string{"Movie"},
		Props:     map[string]any{"title": "Heat", "embedding": floats(1536)},
	}
	other := neo4j.Node{
		ElementId: "4:db:2",
		Labels:    []string{"Movie"},
		Props:     map[string]any{"title": "Ronin", "embedding": floats(1536)},
	}
	similar := neo4j.Relationship{
		ElementId:      "5:db:3",
		StartElementId: "4:db:1",
		EndElementId:   "4:db:2",
		Type:           "SIMILAR",
		Props:          map[string]any{"score": 0.9, "embedding": floats(768)},
	}

	wantMovie := map[string]any{
		"Id":        float64(0),
		"ElementId": "4:db:1",
		"Labels":    []any{"Movie"},
		"Props":     map[string]any{"title": "Heat", "embedding": "<embedding: 1536 values omitted>"},
	}
	wantOther := map[string]any{
		"Id":        float64(0),
		"ElementId": "4:db:2",
		"Labels":    []any{"Movie"},
		"Props":     map[string]any{"title": "Ronin", "embedding": "<embedding: 1536 values omitted>"},
	}
	wantSimilar := map[string]any{
		"Id":             float64(0),
		"ElementId":      "5:db:3",
		"StartId":        float64(0),
		"StartElementId": "4:db:1",
		"EndId":          float64(0),
		"EndElementId":   "4:db:2",
		"Type":           "SIMILAR",
		"Props":          map[string]any{"score": 0.9, "embedding": "<embedding: 768 values omitted>"},
	}

	tests := []struct {
		name  string
		value any
		want  any
	}{
		{name: "node", value: movie, want: wantMovie},
		{name: "relationship", value: similar, want: wantSimilar},
		{
			name:  "path",
			value: neo4j.Path{Nodes: []neo4j.Node{movie, other}, Relationships: []neo4j.Relationship{similar}},
			want: map[string]any{
				"Nodes":         []any{wantMovie, wantOther},
				"Relationships": []any{wantSimilar},
			},
		},
		{
			name:  "list of nodes from collect(n)",
			value: []any{movie, other},
			want:  []any{wantMovie, wantOther},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := recordsToMaps(t, newTestRecord([]string{"x"}, []any{tt.value}))
			if !reflect.DeepEqual(got[0]["x"], tt.want) {
				t.Fatalf("x = %#v\nwant %#v", got[0]["x"], tt.want)
			}
		})
	}
}

// The driver's records must not be modified: they belong to the caller, and a node's Props map
// is shared by every copy of the Node value.
func TestNeo4jRecordsToJSON_DoesNotModifyRecords(t *testing.T) {
	node := neo4j.Node{ElementId: "4:db:1", Props: map[string]any{"embedding": floats(1536)}}
	record := newTestRecord([]string{"n", "v"}, []any{node, floats(1536)})

	recordsToMaps(t, record)

	if got, ok := node.Props["embedding"].([]any); !ok || len(got) != 1536 {
		t.Fatalf("node embedding was modified: %#v", node.Props["embedding"])
	}
	if got, ok := record.Values[1].([]any); !ok || len(got) != 1536 {
		t.Fatalf("record value was modified: %#v", record.Values[1])
	}
}

// Neo4j's native VECTOR type reaches the driver as neo4j.Vector[T], not as a list.
func TestNeo4jRecordsToJSON_OmitsNativeVectors(t *testing.T) {
	tests := []struct {
		name  string
		value any
		want  any
	}{
		{
			name:  "FLOAT32 vector",
			value: neo4j.Vector[float32]{Elems: make([]float32, 1536)},
			want:  "<embedding: 1536 values omitted>",
		},
		{
			name:  "INTEGER8 vector",
			value: neo4j.Vector[int8]{Elems: make([]int8, 1024)},
			want:  "<embedding: 1024 values omitted>",
		},
		{
			name: "vector property on a node",
			value: neo4j.Node{
				ElementId: "4:db:1",
				Labels:    []string{"Chunk"},
				Props:     map[string]any{"text": "a", "embedding": neo4j.Vector[float64]{Elems: make([]float64, 768)}},
			},
			want: map[string]any{
				"Id":        float64(0),
				"ElementId": "4:db:1",
				"Labels":    []any{"Chunk"},
				"Props":     map[string]any{"text": "a", "embedding": "<embedding: 768 values omitted>"},
			},
		},
		{
			name:  "short vector is kept",
			value: neo4j.Vector[float64]{Elems: []float64{0.5, 1.5, 2.5}},
			want:  map[string]any{"Elems": []any{0.5, 1.5, 2.5}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := recordsToMaps(t, newTestRecord([]string{"x"}, []any{tt.value}))
			if !reflect.DeepEqual(got[0]["x"], tt.want) {
				t.Fatalf("x = %#v\nwant %#v", got[0]["x"], tt.want)
			}
		})
	}
}
