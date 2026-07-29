// Copyright (c) "Neo4j"
// Neo4j Sweden AB [http://neo4j.com]

package server

import (
	"net/url"
	"strings"

	"github.com/neo4j/mcp/internal/logger"
)

// TargetInfo holds safe-to-log Neo4j connection metadata derived from a Bolt URI.
type TargetInfo struct {
	Target string
	DBID   string
}

// TargetInfoFromURI builds neo4j_target as scheme://host (no userinfo, no query) and extracts db_id when present.
func TargetInfoFromURI(raw string) (TargetInfo, error) {
	parsed, err := url.Parse(raw)
	if err != nil {
		return TargetInfo{}, err
	}

	return TargetInfo{
		Target: logger.SafeBoltTarget(raw),
		DBID:   dbIDFromHostname(parsed.Hostname()),
	}, nil
}

// dbIDFromHostname returns the Aura database instance id from the hostname first label.
func dbIDFromHostname(host string) string {
	if !isAuraDatabaseHost(host) {
		return ""
	}
	labels := strings.Split(host, ".")
	if len(labels) == 0 {
		return ""
	}
	return labels[0]
}

func isAuraDatabaseHost(host string) bool {
	return strings.HasSuffix(host, ".databases.neo4j.io") ||
		strings.HasSuffix(host, ".databases.neo4j-dev.io")
}
