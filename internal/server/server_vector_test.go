// Copyright (c) "Neo4j"
// Neo4j Sweden AB [http://neo4j.com]

package server

import (
	"testing"

	analytics "github.com/neo4j/mcp/internal/analytics/mocks"
	"github.com/neo4j/mcp/internal/config"
	db "github.com/neo4j/mcp/internal/database/mocks"
	"go.uber.org/mock/gomock"
)

func TestSupportsSearchClause(t *testing.T) {
	cases := []struct {
		version string
		want    bool
	}{
		{"2026.01.0", true},
		{"2026.04", true},
		{"2026.12.1", true},
		{"2027.1.0", true},
		{"2025.11", false},
		{"2025.01.0", false},
		{"5.26.0", false},
		{"5.18.0", false},
		{"4.4.0", false},
		{"unknown", false},
		{"", false},
		{"v2026.01", false}, // leading non-digit => unparseable => false
	}
	for _, c := range cases {
		if got := supportsSearchClause(c.version); got != c.want {
			t.Errorf("supportsSearchClause(%q) = %v, want %v", c.version, got, c.want)
		}
	}
}

func TestSupportsAiTextEmbed(t *testing.T) {
	cases := []struct {
		version string
		want    bool
	}{
		{"5.27-aura", false},
		{"5.26", false},
		{"2024.12", false},
		{"2025.01", false},
		{"2025.10", false},
		{"2025.11", true},
		{"2025.12", true},
		{"2026.01", true},
		{"2027.05", true},
		{"2025", false},
		{"", false},
		{"garbage", false},
	}
	for _, c := range cases {
		if got := supportsAiTextEmbed(c.version); got != c.want {
			t.Errorf("supportsAiTextEmbed(%q) = %v, want %v", c.version, got, c.want)
		}
	}
}

// toolNames returns the registered tool names as a set for readable assertions.
func toolNames(s *Neo4jMCPServer) map[string]bool {
	names := make(map[string]bool)
	for name := range s.MCPServer.ListTools() {
		names[name] = true
	}
	return names
}

// newEmbeddingConfiguredServer builds a server whose config has embedding fully
// configured (OpenAI), for the given transport mode. The DB/analytics services are
// mocked with no expectations because registerTools/addVectorTools perform no DB or
// analytics calls.
func newEmbeddingConfiguredServer(t *testing.T, transport config.TransportMode) (*Neo4jMCPServer, *gomock.Controller) {
	t.Helper()
	t.Setenv("NEO4J_URI", "bolt://test-host:7687")
	t.Setenv("NEO4J_TRANSPORT_MODE", string(transport))
	if transport == config.TransportModeStdio {
		t.Setenv("NEO4J_USERNAME", "neo4j")
		t.Setenv("NEO4J_PASSWORD", "password")
	} else {
		// HTTP mode must not have static credentials.
		t.Setenv("NEO4J_USERNAME", "")
		t.Setenv("NEO4J_PASSWORD", "")
	}
	t.Setenv("NEO4J_EMBEDDING_PROVIDER", config.EmbeddingProviderOpenAI)
	t.Setenv("NEO4J_EMBEDDING_MODEL", "text-embedding-3-small")
	t.Setenv("NEO4J_EMBEDDING_API_KEY", "test-key")

	cfg, err := config.LoadConfig(nil)
	if err != nil {
		t.Fatalf("LoadConfig() failed: %v", err)
	}
	if !cfg.IsEmbeddingConfigured() {
		t.Fatalf("expected embedding to be configured")
	}

	ctrl := gomock.NewController(t)
	mockDB := db.NewMockService(ctrl)
	aService := analytics.NewMockService(ctrl)
	return NewNeo4jMCPServer("test-version", cfg, mockDB, aService), ctrl
}

func TestVectorToolRegistrationStdioConfigured(t *testing.T) {
	s, ctrl := newEmbeddingConfiguredServer(t, config.TransportModeStdio)
	defer ctrl.Finish()

	if err := s.registerTools(); err != nil {
		t.Fatalf("registerTools() failed: %v", err)
	}
	if names := toolNames(s); !names["vector-search"] {
		t.Errorf("expected vector-search to be registered in stdio mode when embedding configured; got %v", names)
	}
}

func TestVectorToolRegisteredInHTTPAtStartup(t *testing.T) {
	s, ctrl := newEmbeddingConfiguredServer(t, config.TransportModeHTTP)
	defer ctrl.Finish()

	if err := s.registerTools(); err != nil {
		t.Fatalf("registerTools() failed: %v", err)
	}
	// vector-search must be present in the INITIAL tool list (registered at startup, not
	// deferred) so clients that import the tool list once — e.g. Copilot Studio — can see
	// it. The version-dependent query strategy is resolved lazily at call time.
	if names := toolNames(s); !names["vector-search"] {
		t.Errorf("expected vector-search to be registered at HTTP startup when embedding configured; got %v", names)
	}
}

func TestVectorToolAbsentWhenEmbeddingNotConfigured(t *testing.T) {
	t.Setenv("NEO4J_URI", "bolt://test-host:7687")
	t.Setenv("NEO4J_TRANSPORT_MODE", string(config.TransportModeStdio))
	t.Setenv("NEO4J_USERNAME", "neo4j")
	t.Setenv("NEO4J_PASSWORD", "password")
	t.Setenv("NEO4J_EMBEDDING_PROVIDER", "") // embedding disabled

	cfg, err := config.LoadConfig(nil)
	if err != nil {
		t.Fatalf("LoadConfig() failed: %v", err)
	}
	if cfg.IsEmbeddingConfigured() {
		t.Fatalf("did not expect embedding to be configured")
	}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	s := NewNeo4jMCPServer("test-version", cfg, db.NewMockService(ctrl), analytics.NewMockService(ctrl))

	if err := s.registerTools(); err != nil {
		t.Fatalf("registerTools() failed: %v", err)
	}
	if names := toolNames(s); names["vector-search"] {
		t.Errorf("expected vector-search to be absent when embedding not configured; got %v", names)
	}
}
