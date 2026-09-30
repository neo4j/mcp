// Copyright (c) "Neo4j"
// Neo4j Sweden AB [http://neo4j.com]

package server

// Per-request HTTP headers understood by the Neo4j MCP server. Exported so that tests
// and other packages reference these constants instead of repeating the wire values,
// which the compiler cannot connect back to here.
const (
	URIHeader      = "X-Neo4j-MCP-URI"
	ToolsHeader    = "X-Neo4j-MCP-Tools"
	ReadOnlyHeader = "X-Neo4j-MCP-ReadOnly"
	TimeoutHeader  = "X-Neo4j-MCP-Request-Timeout"
)
