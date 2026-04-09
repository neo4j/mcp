package server

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
)

// isUnauthenticatedMethodRequest reads the JSON-RPC body and returns true if
// the request is a POST whose "method" field matches the given jsonRPCMethod.
// The body is always restored so downstream handlers can read it normally.
// Caller must have already wrapped r.Body with http.MaxBytesReader.
func isUnauthenticatedMethodRequest(r *http.Request, jsonRPCMethod string) (bool, error) {
	if r.Method != http.MethodPost {
		return false, nil
	}
	if r.ContentLength >= 0 && r.ContentLength > maxUnauthenticatedBodyBytes {
		return false, errRequestBodyTooLarge
	}

	buf, err := io.ReadAll(r.Body)
	// Close the original body to free resources.
	if rc := r.Body; rc != nil {
		_ = rc.Close()
	}

	if err != nil {
		// Replace body with an empty reader to avoid further reads.
		r.Body = io.NopCloser(bytes.NewReader(nil))

		// If MaxBytesReader triggered, it typically returns an error containing
		// "request body too large". Map that to a sentinel error so middleware can
		// respond with 413.
		if strings.Contains(err.Error(), "request body too large") {
			return false, errRequestBodyTooLarge
		}

		return false, err
	}

	// Restore the read bytes so downstream handlers can read the body as usual.
	r.Body = io.NopCloser(bytes.NewReader(buf))

	var probe struct {
		Method string `json:"method"`
	}
	if e := json.Unmarshal(buf, &probe); e != nil {
		return false, e
	}

	return probe.Method == jsonRPCMethod, nil
}

// extractDatabaseFromPath checks if the URL path matches the expected format for database-specific endpoints (/db/{databaseName}/mcp)
func extractDatabaseFromPath(path string) (string, bool) {
	// Expected path format with database name: /db/{databaseName}/*
	parts := strings.Split(path, "/")
	// ["", "db", "{name}", "*"]
	if len(parts) >= 4 && parts[1] == "db" {
		return parts[2], true
	}
	return "", false
}

// isAlphanumeric checks if a character is an ASCII letter or digit
func isAlphanumeric(ch rune) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9')
}

// isValidDatabaseName checks if the provided database name is valid according to Neo4j naming rules
// Reference: https://neo4j.com/docs/cypher-manual/current/administration/databases/standard-databases/naming-rules/
func isValidDatabaseName(name string) bool {
	// Length must be between 3 and 63 characters
	if len(name) < 3 || len(name) > 63 {
		return false
	}

	// Names starting with underscore or "system" are reserved for internal use
	if strings.HasPrefix(name, "_") || strings.HasPrefix(strings.ToLower(name), "system") {
		return false
	}

	// First and last characters must be ASCII alphabetic or numeric
	if !isAlphanumeric(rune(name[0])) || !isAlphanumeric(rune(name[len(name)-1])) {
		return false
	}

	// Subsequent characters must be ASCII alphabetic or numeric, dots, or dashes
	for _, ch := range name[1 : len(name)-1] {
		if !isAlphanumeric(ch) && ch != '.' && ch != '-' {
			return false
		}
	}

	return true
}

// isValidDatabasePath checks if the URL path matches the expected format for MCP endpoints
func isValidDatabasePath(path string) bool {
	// Expected path format: /db/{databaseName}/mcp or /mcp
	parts := strings.Split(path, "/")
	// ["", "db", "{name}", "mcp", +anything else]
	if len(parts) >= 4 && parts[1] == "db" && parts[3] == "mcp" {
		return true
	}

	if len(parts) == 2 && parts[1] == "mcp" { // /mcp (without trailing slash)
		return true
	}

	if len(parts) == 3 && parts[1] == "mcp" && parts[2] == "" { // /mcp/ (trailing slash yields ["", "mcp", ""])
		return true
	}

	return false
}
