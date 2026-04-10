// Copyright (c) "Neo4j"
// Neo4j Sweden AB [http://neo4j.com]

package server

import "testing"

func TestExtractDatabaseFromPath(t *testing.T) {
	tests := []struct {
		name   string
		path   string
		want   string
		wantOK bool
	}{
		{
			name:   "valid path with database",
			path:   "/db/testdb/mcp",
			want:   "testdb",
			wantOK: true,
		},
		{
			name:   "invalid path missing db segment",
			path:   "/testdb/mcp",
			want:   "",
			wantOK: false,
		},
		{
			name:   "3 parts after split should return false",
			path:   "/db/userdb",
			wantOK: false,
		},
		{
			name:   "/db/ should return false",
			path:   "/db/",
			wantOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path, ok := extractDatabaseFromPath(tt.path)
			if path != tt.want {
				t.Errorf("extractDatabaseFromPath() = %v, want %v", path, tt.want)
			}
			if ok != tt.wantOK {
				t.Errorf("extractDatabaseFromPath() = %v, want %v", ok, tt.wantOK)
			}
		})
	}
}

func TestIsAlphanumeric(t *testing.T) {
	tests := []struct {
		name string
		ch   rune
		want bool
	}{
		{
			name: "lowercase letter",
			ch:   'a',
			want: true,
		},
		{
			name: "uppercase letter",
			ch:   'Z',
			want: true,
		},
		{
			name: "digit",
			ch:   '5',
			want: true,
		},
		{
			name: "special character",
			ch:   '@',
			want: false,
		},
		{
			name: "space character",
			ch:   ' ',
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isAlphanumeric(tt.ch)
			if got != tt.want {
				t.Errorf("isAlphanumeric() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsValidDatabaseName(t *testing.T) {
	tests := []struct {
		name   string
		dbName string
		want   bool
	}{
		{
			name:   "valid database name",
			dbName: "mydb",
			want:   true,
		},
		// Length must be between 3 and 63 characters
		{
			name:   "len < 3 should be invalid",
			dbName: "ab",
			want:   false,
		},
		{
			name:   "len > 63 should be invalid",
			dbName: "a-very-long-database-name-that-exceeds-the-maximum-length-limit-of-sixty-three-characters",
			want:   false,
		},
		// Names starting with underscore or "system" are reserved for internal use
		{
			name:   "name starting with underscore should be invalid",
			dbName: "_internaldb",
			want:   false,
		},
		{
			name:   "name starting with 'system' should be invalid",
			dbName: "systemdb",
			want:   false,
		},
		{
			name:   "name starting with 'System' (case-insensitive) should be invalid",
			dbName: "Systemdb",
			want:   false,
		},
		// First and last characters must be ASCII alphabetic or numeric
		{
			name:   "name starting with non-alphanumeric character should be invalid",
			dbName: "-mydb",
			want:   false,
		},
		{
			name:   "name ending with non-alphanumeric character should be invalid",
			dbName: "mydb-",
			want:   false,
		},
		// Subsequent characters must be ASCII alphabetic or numeric, dots, or dashes
		{
			name:   "name with invalid characters should be invalid",
			dbName: "my$db",
			want:   false,
		},
		{
			name:   "name with spaces should be invalid",
			dbName: "my db",
			want:   false,
		},
		{
			name:   "name with valid characters should be valid",
			dbName: "my.db-name",
			want:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isValidDatabaseName(tt.dbName)
			if got != tt.want {
				t.Errorf("isValidDatabaseName() = %v, want %v", got, tt.want)
			}
		})
	}
}
