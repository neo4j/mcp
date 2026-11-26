// Copyright (c) "Neo4j"
// Neo4j Sweden AB [http://neo4j.com]

package cypher

import (
	"reflect"
	"testing"

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

// omitOne runs omitEmbeddings on a single-value record and returns the sanitized value.
func omitOne(t *testing.T, v any) any {
	t.Helper()
	got := omitEmbeddings([]*neo4j.Record{{Keys: []string{"x"}, Values: []any{v}}})
	if len(got) != 1 || len(got[0].Values) != 1 {
		t.Fatalf("omitEmbeddings() = %#v, want one record with one value", got)
	}
	return got[0].Values[0]
}

func TestOmitEmbeddings_Lists(t *testing.T) {
	titles := make([]any, 300)
	ids := make([]any, 300)
	for i := range titles {
		titles[i] = "Heat"
		ids[i] = int64(42)
	}
	mixed := floats(300)
	mixed[299] = nil

	tests := []struct {
		name  string
		value any
		want  any
	}{
		{name: "list of 1536 floats is replaced", value: floats(1536), want: "<embedding: 1536 values omitted>"},
		{name: "list of 128 floats is replaced", value: floats(128), want: "<embedding: 128 values omitted>"},
		{name: "list of 127 floats is kept", value: floats(127), want: floats(127)},
		// Long lists that are not all floats are results the caller asked for, e.g. collect(m.title).
		{name: "long list of strings is kept", value: titles, want: titles},
		{name: "long list of integers is kept", value: ids, want: ids},
		{name: "long list of floats with a null is kept", value: mixed, want: mixed},
		{
			name:  "embedding inside a map is replaced and sibling keys are kept",
			value: map[string]any{"title": "Heat", "embedding": floats(1536)},
			want:  map[string]any{"title": "Heat", "embedding": "<embedding: 1536 values omitted>"},
		},
		{
			name:  "embeddings inside a short list of maps are replaced",
			value: []any{map[string]any{"embedding": floats(1536)}, map[string]any{"embedding": floats(768)}},
			want: []any{
				map[string]any{"embedding": "<embedding: 1536 values omitted>"},
				map[string]any{"embedding": "<embedding: 768 values omitted>"},
			},
		},
		{name: "scalar is kept", value: "Heat", want: "Heat"},
		{name: "null is kept", value: nil, want: nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := omitOne(t, tt.value); !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("got %#v\nwant %#v", got, tt.want)
			}
		})
	}
}

func TestOmitEmbeddings_GraphEntities(t *testing.T) {
	movie := neo4j.Node{ElementId: "4:db:1", Labels: []string{"Movie"}, Props: map[string]any{"title": "Heat", "embedding": floats(1536)}}
	other := neo4j.Node{ElementId: "4:db:2", Labels: []string{"Movie"}, Props: map[string]any{"title": "Ronin", "embedding": floats(1536)}}
	similar := neo4j.Relationship{
		ElementId:      "5:db:3",
		StartElementId: "4:db:1",
		EndElementId:   "4:db:2",
		Type:           "SIMILAR",
		Props:          map[string]any{"score": 0.9, "embedding": floats(768)},
	}

	wantMovie := neo4j.Node{ElementId: "4:db:1", Labels: []string{"Movie"}, Props: map[string]any{"title": "Heat", "embedding": "<embedding: 1536 values omitted>"}}
	wantOther := neo4j.Node{ElementId: "4:db:2", Labels: []string{"Movie"}, Props: map[string]any{"title": "Ronin", "embedding": "<embedding: 1536 values omitted>"}}
	wantSimilar := similar
	wantSimilar.Props = map[string]any{"score": 0.9, "embedding": "<embedding: 768 values omitted>"}

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
			want:  neo4j.Path{Nodes: []neo4j.Node{wantMovie, wantOther}, Relationships: []neo4j.Relationship{wantSimilar}},
		},
		{name: "list of nodes from collect(n)", value: []any{movie, other}, want: []any{wantMovie, wantOther}},
		{name: "node without properties", value: neo4j.Node{ElementId: "4:db:9"}, want: neo4j.Node{ElementId: "4:db:9"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := omitOne(t, tt.value); !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("got %#v\nwant %#v", got, tt.want)
			}
		})
	}
}

// Neo4j's native VECTOR type reaches the driver as neo4j.Vector[T], not as a list.
func TestOmitEmbeddings_NativeVectors(t *testing.T) {
	short := neo4j.Vector[float64]{Elems: []float64{0.5, 1.5, 2.5}}
	tests := []struct {
		name  string
		value any
		want  any
	}{
		{name: "FLOAT32 vector", value: neo4j.Vector[float32]{Elems: make([]float32, 1536)}, want: "<embedding: 1536 values omitted>"},
		{name: "INTEGER8 vector", value: neo4j.Vector[int8]{Elems: make([]int8, 1024)}, want: "<embedding: 1024 values omitted>"},
		{
			name:  "vector property on a node",
			value: neo4j.Node{ElementId: "4:db:1", Props: map[string]any{"text": "a", "embedding": neo4j.Vector[float64]{Elems: make([]float64, 768)}}},
			want:  neo4j.Node{ElementId: "4:db:1", Props: map[string]any{"text": "a", "embedding": "<embedding: 768 values omitted>"}},
		},
		{name: "short vector is kept", value: short, want: short},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := omitOne(t, tt.value); !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("got %#v\nwant %#v", got, tt.want)
			}
		})
	}
}

// The driver's records must not be modified: a node's Props map is shared by every copy of the Node value.
func TestOmitEmbeddings_DoesNotModifyRecords(t *testing.T) {
	node := neo4j.Node{ElementId: "4:db:1", Props: map[string]any{"embedding": floats(1536)}}
	record := &neo4j.Record{Keys: []string{"n", "v"}, Values: []any{node, floats(1536)}}

	got := omitEmbeddings([]*neo4j.Record{record})

	if got[0] == record {
		t.Fatal("omitEmbeddings returned the input record instead of a copy")
	}
	if !reflect.DeepEqual(got[0].Keys, record.Keys) {
		t.Fatalf("keys = %v, want %v", got[0].Keys, record.Keys)
	}
	if v, ok := node.Props["embedding"].([]any); !ok || len(v) != 1536 {
		t.Fatalf("node embedding was modified: %#v", node.Props["embedding"])
	}
	if v, ok := record.Values[1].([]any); !ok || len(v) != 1536 {
		t.Fatalf("record value was modified: %#v", record.Values[1])
	}
}
