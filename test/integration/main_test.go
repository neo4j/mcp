// Copyright (c) "Neo4j"
// Neo4j Sweden AB [http://neo4j.com]

//go:build integration

package integration

import (
	"context"
	"os"
	"testing"

	"github.com/neo4j/mcp/test/testdb"
)

func TestMain(m *testing.M) {
	ctx := context.Background()

	dbs := testdb.GetInstance()
	dbs.Start(ctx)
	code := m.Run()
	dbs.Stop(ctx)

	os.Exit(code)
}
