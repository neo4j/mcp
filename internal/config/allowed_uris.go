// Copyright (c) "Neo4j"
// Neo4j Sweden AB [http://neo4j.com]

package config

import (
	"fmt"
	"net"
	"net/netip"
	"net/url"
	"strconv"
	"strings"
)

const defaultBoltPort = "7687"

// AllowedURI is a single host[:port] entry of NEO4J_MCP_HTTP_ALLOWED_URIS.
// For wildcard entries host holds the suffix including the leading dot (e.g. ".databases.neo4j.io").
type AllowedURI struct {
	host     string
	port     string
	wildcard bool
}

// AllowedURIs restricts the Neo4j instances the server may connect to in HTTP mode.
type AllowedURIs []AllowedURI

// ParseAllowedURIs parses a comma-separated list of host[:port] entries.
// Wildcards are only allowed as a full leading label (e.g. *.databases.neo4j.io). The port defaults to 7687.
func ParseAllowedURIs(raw string) (AllowedURIs, error) {
	entries := ParseCommaSeparatedString(raw)
	allowed := make(AllowedURIs, 0, len(entries))
	for _, entry := range entries {
		allowedURI, err := parseAllowedURI(entry)
		if err != nil {
			return nil, fmt.Errorf("invalid entry %q: %w", entry, err)
		}
		allowed = append(allowed, allowedURI)
	}
	return allowed, nil
}

func parseAllowedURI(entry string) (AllowedURI, error) {
	if u, err := url.Parse(entry); err == nil && u.Scheme != "" && u.Host != "" {
		return AllowedURI{}, fmt.Errorf("scheme %q is not supported, use %q instead", u.Scheme, u.Host)
	}
	if strings.ContainsAny(entry, "/@?#%") {
		return AllowedURI{}, fmt.Errorf("expected host[:port] without scheme, path or userinfo")
	}

	host, port, err := net.SplitHostPort(entry)
	if err != nil {
		host, port = entry, defaultBoltPort
		if strings.HasPrefix(host, "[") && strings.HasSuffix(host, "]") {
			host = host[1 : len(host)-1]
		}
	}

	portNum, err := strconv.Atoi(port)
	if err != nil || portNum < 1 || portNum > 65535 {
		return AllowedURI{}, fmt.Errorf("port must be a number between 1 and 65535")
	}

	host = strings.ToLower(host)
	if suffix, ok := strings.CutPrefix(host, "*."); ok {
		if !strings.Contains(suffix, ".") || !isValidHostname(suffix) {
			return AllowedURI{}, fmt.Errorf("wildcard must be a leading \"*.\" followed by at least two labels")
		}
		return AllowedURI{host: "." + suffix, port: strconv.Itoa(portNum), wildcard: true}, nil
	}

	if _, err := netip.ParseAddr(host); err != nil && !isValidHostname(host) {
		return AllowedURI{}, fmt.Errorf("host must be a hostname, an IP address or a leading wildcard")
	}
	return AllowedURI{host: host, port: strconv.Itoa(portNum)}, nil
}

func isValidHostname(host string) bool {
	for label := range strings.SplitSeq(host, ".") {
		if label == "" || len(label) > 63 || label[0] == '-' || label[len(label)-1] == '-' {
			return false
		}
		for _, c := range label {
			if (c < 'a' || c > 'z') && (c < '0' || c > '9') && c != '-' {
				return false
			}
		}
	}
	return true
}

// Allows reports whether host and port match an entry. An empty port is treated as 7687.
func (l AllowedURIs) Allows(host, port string) bool {
	host = strings.ToLower(host)
	if port == "" {
		port = defaultBoltPort
	}
	for _, allowed := range l {
		if allowed.port != port {
			continue
		}
		if allowed.wildcard && strings.HasSuffix(host, allowed.host) || host == allowed.host {
			return true
		}
	}
	return false
}
