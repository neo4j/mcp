// Copyright (c) "Neo4j"
// Neo4j Sweden AB [http://neo4j.com]

package mcpcontext

import (
	"context"
	"testing"
)

func TestBearerToken(t *testing.T) {
	tests := []struct {
		name      string
		setupCtx  context.Context
		wantToken string
		wantOK    bool
	}{
		{
			name:      "token stored and retrieved",
			setupCtx:  WithBearerToken(context.Background(), "test-bearer-token"),
			wantToken: "test-bearer-token",
			wantOK:    true,
		},
		{
			name:     "missing token returns empty and false",
			setupCtx: context.Background(),
			wantOK:   false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			token, ok := GetBearerToken(tc.setupCtx)
			if ok != tc.wantOK {
				t.Errorf("ok: got %v, want %v", ok, tc.wantOK)
			}
			if token != tc.wantToken {
				t.Errorf("token: got %q, want %q", token, tc.wantToken)
			}
		})
	}
}

func TestBasicAuth(t *testing.T) {
	tests := []struct {
		name     string
		setupCtx context.Context
		wantUser string
		wantPass string
		wantOK   bool
	}{
		{
			name:     "credentials stored and retrieved",
			setupCtx: WithBasicAuth(context.Background(), "testuser", "testpass"),
			wantUser: "testuser",
			wantPass: "testpass",
			wantOK:   true,
		},
		{
			name:     "missing credentials return empty and false",
			setupCtx: context.Background(),
			wantOK:   false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			user, pass, ok := GetBasicAuthCredentials(tc.setupCtx)
			if ok != tc.wantOK {
				t.Errorf("ok: got %v, want %v", ok, tc.wantOK)
			}
			if user != tc.wantUser {
				t.Errorf("user: got %q, want %q", user, tc.wantUser)
			}
			if pass != tc.wantPass {
				t.Errorf("pass: got %q, want %q", pass, tc.wantPass)
			}
		})
	}
}

func TestHasAuth(t *testing.T) {
	tests := []struct {
		name     string
		setupCtx context.Context
		want     bool
	}{
		{
			name:     "with basic auth",
			setupCtx: WithBasicAuth(context.Background(), "user", "pass"),
			want:     true,
		},
		{
			name:     "with bearer token",
			setupCtx: WithBearerToken(context.Background(), "token"),
			want:     true,
		},
		{
			name:     "with no auth",
			setupCtx: context.Background(),
			want:     false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := HasAuth(tc.setupCtx); got != tc.want {
				t.Errorf("HasAuth: got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestDatabaseName(t *testing.T) {
	tests := []struct {
		name     string
		setupCtx context.Context
		wantName string
		wantOK   bool
	}{
		{
			name:     "database name stored and retrieved",
			setupCtx: WithDatabaseName(context.Background(), "user-db"),
			wantName: "user-db",
			wantOK:   true,
		},
		{
			name:     "missing database name returns empty and false",
			setupCtx: context.Background(),
			wantOK:   false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			name, ok := GetDatabaseName(tc.setupCtx)
			if ok != tc.wantOK {
				t.Errorf("ok: got %v, want %v", ok, tc.wantOK)
			}
			if name != tc.wantName {
				t.Errorf("name: got %q, want %q", name, tc.wantName)
			}
		})
	}
}
