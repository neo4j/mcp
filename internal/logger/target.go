// Copyright (c) "Neo4j"
// Neo4j Sweden AB [http://neo4j.com]

package logger

import "net/url"

// SafeBoltTarget returns scheme://host from a Bolt URI, excluding userinfo and query parameters.
// Log the result as neo4j_target, never with denylisted keys such as bolt_uri or uri.
func SafeBoltTarget(raw string) string {
	parsed, err := url.Parse(raw)
	if err != nil {
		return "[invalid URI]"
	}
	return parsed.Scheme + "://" + parsed.Host
}
