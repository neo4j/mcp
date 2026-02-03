//go:build integration

package integration

import (
	"context"
	"testing"

	"github.com/neo4j/mcp/test/dbservice"
)

var testDbs = dbservice.NewDBService()

func TestIfWeStillSupportVersionOne(t *testing.T) {
	ctx := context.Background()

	t.Run("version 1.0 check", func(t *testing.T) {
		testDbs.Start(ctx, "1.0")

		testDbs.Stop(ctx)
	})
}
