//go:build e2e

package e2e

import (
	"context"
	"log"
	"os"
	"testing"

	"github.com/neo4j/mcp/test/dbservice"
	"github.com/neo4j/mcp/test/e2e/helpers"
)

var dbs = dbservice.NewDBService()
var server string = ""

func TestMain(m *testing.M) {
	ctx := context.Background()

	srv, cleanUpServerDir, err := helpers.BuildServer()
	server = srv

	if err != nil {
		log.Fatal("error while creating MCP server for e2e purpose")
	}
	dbs.Start(ctx, "latest")

	code := m.Run()

	dbs.Stop(ctx)

	cleanUpServerDir()

	os.Exit(code)
}
