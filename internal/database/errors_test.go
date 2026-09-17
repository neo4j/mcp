// Copyright (c) "Neo4j"
// Neo4j Sweden AB [http://neo4j.com]

package database

import (
	"errors"
	"testing"

	"github.com/neo4j/neo4j-go-driver/v6/neo4j"
)

func TestNeo4jErrorCode(t *testing.T) {
	t.Run("returns code for Neo4jError", func(t *testing.T) {
		err := &neo4j.Neo4jError{Code: "Neo.ClientError.Security.Unauthorized", Msg: "unauthorized"}
		if got := Neo4jErrorCode(err); got != "Neo.ClientError.Security.Unauthorized" {
			t.Fatalf("expected Neo4j error code, got %q", got)
		}
	})

	t.Run("returns empty for other errors", func(t *testing.T) {
		if got := Neo4jErrorCode(errors.New("other")); got != "" {
			t.Fatalf("expected empty code, got %q", got)
		}
	})
}
