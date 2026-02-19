//
// Copyright (c) "Neo4j"
// Neo4j Sweden AB [http://neo4j.com]

package database_test

import (
	"testing"

	"github.com/neo4j/mcp/internal/database"
	"github.com/neo4j/neo4j-go-driver/v6/neo4j"
)

// helper to construct a *neo4j.Record for testing using public fields
func newTestRecord(keys []string, values []any) *neo4j.Record {
	return &neo4j.Record{
		Keys:   keys,
		Values: values,
	}
}

func TestNeo4jService_Neo4jRecordsToJSON(t *testing.T) {
	var nilRecords []*neo4j.Record // this is a nil slice

	tests := []struct {
		name    string
		records []*neo4j.Record
		want    string
		wantErr bool
	}{
		{
			name:    "nil slice returns empty JSON array",
			records: nilRecords,
			want:    "[]",
			wantErr: false,
		},
		{
			name:    "empty slice returns empty JSON array",
			records: []*neo4j.Record{},
			want:    "[]",
			wantErr: false,
		},
		{
			name: "single record with valid data",
			records: []*neo4j.Record{
				newTestRecord([]string{"name", "age"}, []any{"Alice", 30}),
			},
			want:    "[\n  {\n    \"age\": 30,\n    \"name\": \"Alice\"\n  }\n]",
			wantErr: false,
		},
		{
			name: "multiple records",
			records: []*neo4j.Record{
				newTestRecord([]string{"name", "age"}, []any{"Alice", 30}),
				newTestRecord([]string{"name", "age"}, []any{"Bob", 25}),
			},
			want:    "[\n  {\n    \"age\": 30,\n    \"name\": \"Alice\"\n  },\n  {\n    \"age\": 25,\n    \"name\": \"Bob\"\n  }\n]",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var s database.Neo4jService
			got, err := s.Neo4jRecordsToJSON(tt.records)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Neo4jRecordsToJSON() error = %v, wantErr %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Fatalf("Neo4jRecordsToJSON() = %q, want %q", got, tt.want)
			}
		})
	}
}
