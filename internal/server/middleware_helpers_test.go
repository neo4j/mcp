// Copyright (c) "Neo4j"
// Neo4j Sweden AB [http://neo4j.com]

package server

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseMCPPath(t *testing.T) {
	tests := []struct {
		name   string
		path   string
		wantDB string
		wantOK bool
	}{
		{
			name:   "/db/{name}/mcp returns database name",
			path:   "/db/testdb/mcp",
			wantDB: "testdb",
			wantOK: true,
		},
		{
			name:   "/db/{name}/mcp/ (trailing slash) returns database name",
			path:   "/db/testdb/mcp/",
			wantDB: "testdb",
			wantOK: true,
		},
		{
			name:   "unrecognised path is invalid",
			path:   "/testdb/mcp",
			wantDB: "",
			wantOK: false,
		},
		{
			name:   "/db/{name} without mcp segment is invalid",
			path:   "/db/userdb",
			wantDB: "",
			wantOK: false,
		},
		{
			name:   "extra segments after mcp are invalid",
			path:   "/db/testdb/mcp/extra",
			wantDB: "",
			wantOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, ok := parseMCPPath(tt.path)
			assert.Equal(t, tt.wantDB, db)
			assert.Equal(t, tt.wantOK, ok)
		})
	}
}

func TestIsAlphanumeric(t *testing.T) {
	tests := []struct {
		name string
		ch   rune
		want bool
	}{
		{name: "lowercase letter", ch: 'a', want: true},
		{name: "uppercase letter", ch: 'Z', want: true},
		{name: "digit", ch: '5', want: true},
		{name: "special character", ch: '@', want: false},
		{name: "space character", ch: ' ', want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, isAlphanumeric(tt.ch))
		})
	}
}

func TestIsValidDatabaseName(t *testing.T) {
	tests := []struct {
		name   string
		dbName string
		want   bool
	}{
		{name: "valid database name", dbName: "mydb", want: true},
		// Length must be between 3 and 63 characters
		{name: "len < 3 should be invalid", dbName: "ab", want: false},
		{name: "len > 63 should be invalid", dbName: "a-very-long-database-name-that-exceeds-the-maximum-length-limit-of-sixty-three-characters", want: false},
		// Names starting with underscore or "system" are reserved for internal use
		{name: "name starting with underscore should be invalid", dbName: "_internaldb", want: false},
		{name: "name starting with 'system' should be invalid", dbName: "systemdb", want: false},
		{name: "name starting with 'System' (case-insensitive) should be invalid", dbName: "Systemdb", want: false},
		// First and last characters must be ASCII alphabetic or numeric
		{name: "name starting with non-alphanumeric character should be invalid", dbName: "-mydb", want: false},
		{name: "name ending with non-alphanumeric character should be invalid", dbName: "mydb-", want: false},
		// Subsequent characters must be ASCII alphabetic or numeric, dots, or dashes
		{name: "name with invalid characters should be invalid", dbName: "my$db", want: false},
		{name: "name with spaces should be invalid", dbName: "my db", want: false},
		{name: "name with valid characters should be valid", dbName: "my.db-name", want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, isValidDatabaseName(tt.dbName))
		})
	}
}
