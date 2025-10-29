//go:build integration

package integration

import (
	"context"
	"os"
	"testing"

	"github.com/neo4j/mcp/test/integration/container_runner"
)

func TestMain(m *testing.M) {
	ctx := context.Background()

	container_runner.Start(ctx)

	code := m.Run()

	container_runner.Close(ctx)

	os.Exit(code)
}
