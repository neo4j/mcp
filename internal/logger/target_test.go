// Copyright (c) "Neo4j"
// Neo4j Sweden AB [http://neo4j.com]

package logger_test

import (
	"testing"

	"github.com/neo4j/mcp/internal/logger"
)

func TestSafeBoltTarget(t *testing.T) {
	got := logger.SafeBoltTarget("bolt://user:pass@host:7687")
	if got != "bolt://host:7687" {
		t.Fatalf("expected userinfo stripped, got %q", got)
	}
}
