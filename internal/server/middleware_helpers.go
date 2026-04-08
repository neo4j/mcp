package server

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"regexp"
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

// isValidDatabaseName checks if the provided database name is valid according to Neo4j naming rules
// (alphanumeric, underscores, hyphens, dots, 1-63 characters)
func isValidDatabaseName(name string) bool {
	var validDatabaseName = regexp.MustCompile(`^[a-zA-Z0-9._-]{1,63}$`)
	return validDatabaseName.MatchString(name)
}

// isValidDatabasePath checks if the URL path matches the expected format for MCP endpoints
func isValidDatabasePath(path string) bool {
	// Expected path format: /db/{databaseName}/mcp or /mcp
	parts := strings.Split(path, "/")
	// ["", "db", "{name}", "mcp", +anything else]
	if len(parts) >= 4 && parts[1] == "db" && (parts[3] == "mcp") {
		return isValidDatabaseName(parts[2])
	}

	if len(parts) == 2 && parts[1] == "mcp" { // /mcp (without trailing slash)
		return true
	}

	if len(parts) == 3 && parts[1] == "mcp" && parts[2] == "" { // /mcp/ (trailing slash yields ["", "mcp", ""])
		return true
	}

	return false
}
