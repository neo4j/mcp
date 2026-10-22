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

func TestParseAllowedURIs_InvalidWithSuggestion(t *testing.T) {
	tests := []struct {
		raw  string
		want string
	}{
		{raw: "bolt://db.example.com", want: `scheme "bolt" is not supported, use "db.example.com" instead`},
		{raw: "neo4j+s://user@abc.databases.neo4j.io:7687/", want: `scheme "neo4j+s" is not supported, use "abc.databases.neo4j.io:7687" instead`},
		{raw: "bolt://[fd00::1]:7687", want: `scheme "bolt" is not supported, use "[fd00::1]:7687" instead`},
	}
	for _, tt := range tests {
		t.Run(tt.raw, func(t *testing.T) {
			_, err := ParseAllowedURIs(tt.raw)
			assert.ErrorContains(t, err, tt.want)
		})
	}
}

func TestAllowedURIs_Allows(t *testing.T) {
	tests := []struct {
		name    string
		allowed string
		host    string
		port    string
		want    bool
	}{
		{
			name:    "exact host and port",
			allowed: "db.example.com",
			host:    "db.example.com",
			port:    "7687",
			want:    true,
		},
		{
			name:    "missing port defaults to 7687",
			allowed: "db.example.com",
			host:    "db.example.com",
			port:    "",
			want:    true,
		},
		{
			name:    "host is case-insensitive",
			allowed: "db.example.com",
			host:    "DB.EXAMPLE.COM",
			port:    "7687",
			want:    true,
		},
		{
			name:    "different port",
			allowed: "db.example.com",
			host:    "db.example.com",
			port:    "7473",
			want:    false,
		},
		{
			name:    "different host",
			allowed: "db.example.com",
			host:    "other.example.com",
			port:    "7687",
			want:    false,
		},
		{
			name:    "any entry of the list matches",
			allowed: "a.example.com,b.example.com",
			host:    "b.example.com",
			port:    "7687",
			want:    true,
		},
		{
			name:    "wildcard matches subdomain",
			allowed: "*.databases.neo4j.io",
			host:    "abc.databases.neo4j.io",
			port:    "7687",
			want:    true,
		},
		{
			name:    "wildcard matches nested subdomain",
			allowed: "*.databases.neo4j.io",
			host:    "a.b.databases.neo4j.io",
			port:    "7687",
			want:    true,
		},
		{
			name:    "wildcard does not match apex",
			allowed: "*.databases.neo4j.io",
			host:    "databases.neo4j.io",
			port:    "7687",
			want:    false,
		},
		{
			name:    "wildcard does not match partial label",
			allowed: "*.databases.neo4j.io",
			host:    "abcdatabases.neo4j.io",
			port:    "7687",
			want:    false,
		},
		{
			name:    "wildcard with different port",
			allowed: "*.databases.neo4j.io",
			host:    "abc.databases.neo4j.io",
			port:    "7473",
			want:    false,
		},
		{
			name:    "non-default port matches",
			allowed: "10.0.0.5:7473",
			host:    "10.0.0.5",
			port:    "7473",
			want:    true,
		},
		{
			name:    "non-default port does not match missing port",
			allowed: "10.0.0.5:7473",
			host:    "10.0.0.5",
			port:    "",
			want:    false,
		},
		{
			name:    "IPv6 matches",
			allowed: "[fd00::1]:7687",
			host:    "fd00::1",
			port:    "7687",
			want:    true,
		},
		{
			name:    "IPv6 different address",
			allowed: "[fd00::1]:7687",
			host:    "fd00::2",
			port:    "7687",
			want:    false,
		},
		{
			name:    "empty list matches nothing",
			allowed: "",
			host:    "db.example.com",
			port:    "7687",
			want:    false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			allowed, err := ParseAllowedURIs(tt.allowed)
			require.NoError(t, err)
			assert.Equal(t, tt.want, allowed.Allows(tt.host, tt.port))
		})
	}
}
