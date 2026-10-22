// Copyright (c) "Neo4j"
// Neo4j Sweden AB [http://neo4j.com]

package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseAllowedURIs(t *testing.T) {
	tests := []struct {
		raw  string
		want AllowedURIs
	}{
		{raw: "db.example.com", want: AllowedURIs{{host: "db.example.com", port: "7687"}}},
		{raw: "db.example.com:7687", want: AllowedURIs{{host: "db.example.com", port: "7687"}}},
		{raw: "DB.Example.com:7473", want: AllowedURIs{{host: "db.example.com", port: "7473"}}},
		{raw: "10.0.0.5", want: AllowedURIs{{host: "10.0.0.5", port: "7687"}}},
		{raw: "10.0.0.5:7687", want: AllowedURIs{{host: "10.0.0.5", port: "7687"}}},
		{raw: "[fd00::1]:7687", want: AllowedURIs{{host: "fd00::1", port: "7687"}}},
		{raw: "[fd00::1]", want: AllowedURIs{{host: "fd00::1", port: "7687"}}},
		{raw: "fd00::1", want: AllowedURIs{{host: "fd00::1", port: "7687"}}},
		{raw: "*.databases.neo4j.io", want: AllowedURIs{{host: ".databases.neo4j.io", port: "7687", wildcard: true}}},
		{raw: "*.databases.neo4j.io:7687", want: AllowedURIs{{host: ".databases.neo4j.io", port: "7687", wildcard: true}}},
		{raw: "a.com,,b.com", want: AllowedURIs{{host: "a.com", port: "7687"}, {host: "b.com", port: "7687"}}},
		{raw: " , ", want: AllowedURIs{}},
	}
	for _, tt := range tests {
		t.Run(tt.raw, func(t *testing.T) {
			got, err := ParseAllowedURIs(tt.raw)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestParseAllowedURIs_Invalid(t *testing.T) {
	for _, raw := range []string{
		"bolt://db.example.com",
		"db.example.com/path",
		"user@db.example.com",
		"10.0.0.0/8",
		"*",
		"*.com",
		"db.*.com",
		"d*.example.com",
		"db.example.com:0",
		"db.example.com:abc",
		"db.example.com:",
		"-db.example.com",
		"[fd00::1",
	} {
		t.Run(raw, func(t *testing.T) {
			_, err := ParseAllowedURIs(raw)
			assert.ErrorContains(t, err, "invalid entry")
		})
	}
}

func TestAllowedURIs_Allows(t *testing.T) {
	allowed, err := ParseAllowedURIs("db.example.com,*.databases.neo4j.io,10.0.0.5:7473,[fd00::1]:7687")
	require.NoError(t, err)

	tests := []struct {
		host string
		port string
		want bool
	}{
		{host: "db.example.com", port: "7687", want: true},
		{host: "db.example.com", port: "", want: true},
		{host: "DB.EXAMPLE.COM", port: "", want: true},
		{host: "db.example.com", port: "7473", want: false},
		{host: "other.example.com", port: "7687", want: false},
		{host: "abc.databases.neo4j.io", port: "", want: true},
		{host: "a.b.databases.neo4j.io", port: "7687", want: true},
		{host: "databases.neo4j.io", port: "7687", want: false},
		{host: "abcdatabases.neo4j.io", port: "7687", want: false},
		{host: "abc.databases.neo4j.io", port: "7473", want: false},
		{host: "10.0.0.5", port: "7473", want: true},
		{host: "10.0.0.5", port: "", want: false},
		{host: "fd00::1", port: "7687", want: true},
		{host: "fd00::2", port: "7687", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.host+":"+tt.port, func(t *testing.T) {
			assert.Equal(t, tt.want, allowed.Allows(tt.host, tt.port))
		})
	}

	assert.False(t, AllowedURIs{}.Allows("db.example.com", "7687"))
}
