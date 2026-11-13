// Copyright (c) "Neo4j"
// Neo4j Sweden AB [http://neo4j.com]

package server

import "testing"

func TestTargetInfoFromURI(t *testing.T) {
	info, err := TargetInfoFromURI("neo4j+s://e0c3fbb3.databases.neo4j.io")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.Target != "neo4j+s://e0c3fbb3.databases.neo4j.io" {
		t.Fatalf("unexpected target: %q", info.Target)
	}
	if info.DBID != "e0c3fbb3" {
		t.Fatalf("expected db id, got %q", info.DBID)
	}
}

func TestDBIDFromHostnameStaging(t *testing.T) {
	if dbID := dbIDFromHostname("e0c3fbb3-staging.databases.neo4j.io"); dbID != "e0c3fbb3-staging" {
		t.Fatalf("expected staging db id, got %q", dbID)
	}
}

func TestDBIDFromHostnameDev(t *testing.T) {
	if dbID := dbIDFromHostname("e0c3fbb3-devmachine.databases.neo4j-dev.io"); dbID != "e0c3fbb3-devmachine" {
		t.Fatalf("expected dev db id, got %q", dbID)
	}
}

func TestDBIDFromHostnameNonAura(t *testing.T) {
	if dbID := dbIDFromHostname("localhost"); dbID != "" {
		t.Fatalf("expected empty db id, got %q", dbID)
	}
	if dbID := dbIDFromHostname("other.neo4j.io"); dbID != "" {
		t.Fatalf("expected empty db id for non-database host, got %q", dbID)
	}
}
