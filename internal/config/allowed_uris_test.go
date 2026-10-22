// Copyright (c) "Neo4j"
// Neo4j Sweden AB [http://neo4j.com]

package config

import (
	"fmt"
	"strings"
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
		{raw: "db_example.com", want: AllowedURIs{{host: "db_example.com", port: "7687"}}},
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
	const (
		formatErr   = "expected host[:port] without scheme, path or userinfo"
		portErr     = "port must be a number between 1 and 65535"
		hostErr     = "host must be a hostname, an IP address or a leading wildcard"
		wildcardErr = `wildcard must be a leading "*." followed by at least two labels`
	)
	longLabel := strings.Repeat("a", 64) + ".example.com"

	tests := []struct {
		raw    string
		entry  string
		reason string
	}{
		{
			raw:    "bolt://db.example.com",
			entry:  "bolt://db.example.com",
			reason: `scheme "bolt" is not supported, use "db.example.com" instead`,
		},
		{
			raw:    "bolt:db.example.com",
			entry:  "bolt:db.example.com",
			reason: portErr,
		},
		{
			raw:    "db.example.com/path",
			entry:  "db.example.com/path",
			reason: formatErr,
		},
		{
			raw:    "user@db.example.com",
			entry:  "user@db.example.com",
			reason: formatErr,
		},
		{
			raw:    "10.0.0.0/8",
			entry:  "10.0.0.0/8",
			reason: formatErr,
		},
		{
			raw:    "db.example.com?x=1",
			entry:  "db.example.com?x=1",
			reason: formatErr,
		},
		{
			raw:    "db.example.com#frag",
			entry:  "db.example.com#frag",
			reason: formatErr,
		},
		{
			raw:    "fe80::1%eth0",
			entry:  "fe80::1%eth0",
			reason: formatErr,
		},
		{
			raw:    "db.example.com:0",
			entry:  "db.example.com:0",
			reason: portErr,
		},
		{
			raw:    "db.example.com:65536",
			entry:  "db.example.com:65536",
			reason: portErr,
		},
		{
			raw:    "db.example.com:abc",
			entry:  "db.example.com:abc",
			reason: portErr,
		},
		{
			raw:    "db.example.com:",
			entry:  "db.example.com:",
			reason: portErr,
		},
		{
			raw:    ":7687",
			entry:  ":7687",
			reason: hostErr,
		},
		{
			raw:    "db.example.com:7687:7687",
			entry:  "db.example.com:7687:7687",
			reason: hostErr,
		},
		{
			raw:    "-db.example.com",
			entry:  "-db.example.com",
			reason: hostErr,
		},
		{
			raw:    "db-.example.com",
			entry:  "db-.example.com",
			reason: hostErr,
		},
		{
			raw:    "db.example.com.",
			entry:  "db.example.com.",
			reason: hostErr,
		},
		{
			raw:    longLabel,
			entry:  longLabel,
			reason: hostErr,
		},
		{
			raw:    "[fd00::1",
			entry:  "[fd00::1",
			reason: hostErr,
		},
		{
			raw:    "*",
			entry:  "*",
			reason: hostErr,
		},
		{
			raw:    "db.*.com",
			entry:  "db.*.com",
			reason: hostErr,
		},
		{
			raw:    "d*.example.com",
			entry:  "d*.example.com",
			reason: hostErr,
		},
		{
			raw:    "*.com",
			entry:  "*.com",
			reason: wildcardErr,
		},
		{
			raw:    "*.*.example.com",
			entry:  "*.*.example.com",
			reason: wildcardErr,
		},
		{
			raw:    "a.com, b.com, c.com:abc",
			entry:  "c.com:abc",
			reason: portErr,
		},
	}
	for _, tt := range tests {
		t.Run(tt.raw, func(t *testing.T) {
			_, err := ParseAllowedURIs(tt.raw)
			assert.EqualError(t, err, fmt.Sprintf("invalid entry %q: %s", tt.entry, tt.reason))
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
			name:    "single-label host",
			allowed: "localhost",
			host:    "localhost",
			port:    "",
			want:    true,
		},
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
