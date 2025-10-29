//go:build integration

package integration

import (
	"context"
	"os"
	"testing"

	"github.com/neo4j/mcp/test/integration/containerrunner"
)

func TestMain(m *testing.M) {
	ctx := context.Background()

	containerrunner.Start(ctx)

	code := m.Run()

	containerrunner.Close(ctx)

	os.Exit(code)
}
