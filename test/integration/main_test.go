//go:build integration

package integration

import (
	"context"
	"os"
	"testing"

	"github.com/neo4j/mcp/test/integration/helpers"
)

func TestMain(m *testing.M) {
	ctx := context.Background()

	helpers.Start(ctx)

	code := m.Run()

	helpers.Close(ctx)

	os.Exit(code)
}
