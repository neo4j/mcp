// Copyright (c) "Neo4j"
// Neo4j Sweden AB [http://neo4j.com]

package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"testing"
)

// envVarPattern matches every NEO4J_MCP_ environment variable name referenced in config.go,
// however it is read (GetEnv, GetEnvWithDefault, os.LookupEnv, or error messages).
var envVarPattern = regexp.MustCompile(`NEO4J_MCP_[A-Z0-9_]+`)

// envVarsExcludedFromRegistryEntry lists environment variables that internal/config reads but
// server.json (the MCP Registry entry) deliberately omits, with the reason. The registry entry
// is the install-time configuration form MCP clients render, so it lists only variables that
// apply to the stdio package it declares and that users choose at install time. Deployment
// knobs and HTTP-transport-only variables belong in the README, not the install form.
var envVarsExcludedFromRegistryEntry = map[string]string{
	"NEO4J_MCP_HTTP_ALLOW_UNAUTHENTICATED_PING": "HTTP transport only; the registry package runs stdio",
	"NEO4J_MCP_HTTP_ALLOWED_ORIGINS":            "HTTP transport only; the registry package runs stdio",
	"NEO4J_MCP_HTTP_ALLOWED_URIS":               "HTTP transport only; the registry package runs stdio",
	"NEO4J_MCP_HTTP_AUTH_HEADER_NAME":           "HTTP transport only; the registry package runs stdio",
	"NEO4J_MCP_HTTP_HOST":                       "HTTP transport only; the registry package runs stdio",
	"NEO4J_MCP_HTTP_PORT":                       "HTTP transport only; the registry package runs stdio",
	"NEO4J_MCP_HTTP_TLS_CERT_FILE":              "HTTP transport only; the registry package runs stdio",
	"NEO4J_MCP_HTTP_TLS_ENABLED":                "HTTP transport only; the registry package runs stdio",
	"NEO4J_MCP_HTTP_TLS_KEY_FILE":               "HTTP transport only; the registry package runs stdio",
	"NEO4J_MCP_LOG_FORMAT":                      "deployment knob, not an install-time choice",
	"NEO4J_MCP_TELEMETRY":                       "deployment knob, not an install-time choice",
	"NEO4J_MCP_TRANSPORT_MODE":                  "transport is fixed to stdio by the registry package's transport declaration",
}

// TestServerJsonListsConfiguredEnvVars fails when an environment variable is added to or
// removed from config.go without server.json (the MCP Registry entry) and
// envVarsExcludedFromRegistryEntry being kept in step. The test is the drift check: a new
// variable must either be listed in server.json or gain an exclusion entry with a reason here.
func TestServerJsonListsConfiguredEnvVars(t *testing.T) {
	_, thisFile, _, _ := runtime.Caller(0)
	dir := filepath.Dir(thisFile)

	source, err := os.ReadFile(filepath.Join(dir, "config.go"))
	if err != nil {
		t.Fatalf("reading config.go: %v", err)
	}
	raw, err := os.ReadFile(filepath.Join(dir, "..", "..", "server.json"))
	if err != nil {
		t.Fatalf("reading server.json: %v", err)
	}

	// Collect every env var name config.go references.
	configured := map[string]bool{}
	for _, name := range envVarPattern.FindAllString(string(source), -1) {
		configured[name] = true
	}

	// Collect the env var names server.json declares across all packages.
	var entry struct {
		Packages []struct {
			EnvironmentVariables []struct {
				Name string `json:"name"`
			} `json:"environmentVariables"`
		} `json:"packages"`
	}
	if err := json.Unmarshal(raw, &entry); err != nil {
		t.Fatalf("parsing server.json: %v", err)
	}
	listed := map[string]bool{}
	var order []string
	for _, pkg := range entry.Packages {
		for _, envVar := range pkg.EnvironmentVariables {
			name := envVar.Name
			if listed[name] {
				t.Errorf("server.json lists %s more than once", name)
			}
			if !configured[name] {
				t.Errorf("server.json lists %s, but config.go no longer reads it; remove it from the entry", name)
			}
			listed[name] = true
			order = append(order, name)
		}
	}

	// Every configured variable is either listed or explicitly excluded with a reason.
	for name := range configured {
		switch {
		case listed[name]:
		case envVarsExcludedFromRegistryEntry[name] != "":
		default:
			t.Errorf("config.go reads %s but server.json does not list it; add it to the registry entry or to envVarsExcludedFromRegistryEntry in server_json_test.go with a reason", name)
		}
	}

	// Exclusion entries must stay real: drop them when the variable disappears.
	for name := range envVarsExcludedFromRegistryEntry {
		if !configured[name] {
			t.Errorf("envVarsExcludedFromRegistryEntry still excludes %s, but config.go no longer reads it; remove the stale entry", name)
		}
		if slices.Contains(order, name) {
			t.Errorf("%s is both listed in server.json and excluded in envVarsExcludedFromRegistryEntry; pick one", name)
		}
	}
}
