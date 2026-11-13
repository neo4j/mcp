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

// parseAllowedURI validates a single entry in three steps: its overall shape, the port, then the host,
// which is either a wildcard, an IP address or a hostname.
func parseAllowedURI(entry string) (AllowedURI, error) {
	if err := checkHostPortShape(entry); err != nil {
		return AllowedURI{}, err
	}

	host, port := splitHostPort(entry)

	port, err := normalizePort(port)
	if err != nil {
		return AllowedURI{}, err
	}

	host = strings.ToLower(host)
	if suffix, ok := strings.CutPrefix(host, "*."); ok {
		if !isValidWildcardSuffix(suffix) {
			return AllowedURI{}, fmt.Errorf("wildcard must be a leading \"*.\" followed by at least two labels")
		}
		return AllowedURI{host: "." + suffix, port: port, wildcard: true}, nil
	}

	if !isIPAddress(host) && !isValidHostname(host) {
		return AllowedURI{}, fmt.Errorf("host must be a hostname, an IP address or a leading wildcard")
	}
	return AllowedURI{host: host, port: port}, nil
}

// checkHostPortShape rejects entries that are full URIs, suggesting the host[:port] to use instead,
// and entries with a path, userinfo, query, fragment or IPv6 zone.
func checkHostPortShape(entry string) error {
	if u, err := url.Parse(entry); err == nil && u.Scheme != "" && u.Host != "" {
		return fmt.Errorf("scheme %q is not supported, use %q instead", u.Scheme, u.Host)
	}
	if strings.ContainsAny(entry, "/@?#%") {
		return fmt.Errorf("expected host[:port] without scheme, path or userinfo")
	}
	return nil
}

// splitHostPort splits host[:port], defaulting the port to 7687.
// Anything net.SplitHostPort cannot split is treated as a bare host (e.g. "fd00::1" or "[fd00::1]")
// and left to the host validation.
func splitHostPort(entry string) (host, port string) {
	host, port, err := net.SplitHostPort(entry)
	if err == nil {
		return host, port
	}
	if strings.HasPrefix(entry, "[") && strings.HasSuffix(entry, "]") {
		return entry[1 : len(entry)-1], defaultBoltPort
	}
	return entry, defaultBoltPort
}

// normalizePort checks that port is between 1 and 65535 and returns it in canonical form (e.g. "07687" becomes "7687").
func normalizePort(port string) (string, error) {
	portNum, err := strconv.Atoi(port)
	if err != nil || portNum < 1 || portNum > 65535 {
		return "", fmt.Errorf("port must be a number between 1 and 65535")
	}
	return strconv.Itoa(portNum), nil
}

// isValidWildcardSuffix reports whether the part after "*." is a hostname with at least two labels, so "*.com" is rejected.
func isValidWildcardSuffix(suffix string) bool {
	return strings.Contains(suffix, ".") && isValidHostname(suffix)
}

func isIPAddress(host string) bool {
	_, err := netip.ParseAddr(host)
	return err == nil
}

// isValidHostname reports whether host is made of dot-separated labels of 1-63 characters from [a-z0-9-_],
// none starting or ending with '-'.
func isValidHostname(host string) bool {
	for label := range strings.SplitSeq(host, ".") {
		if label == "" || len(label) > 63 || label[0] == '-' || label[len(label)-1] == '-' {
			return false
		}
		for _, c := range label {
			if (c < 'a' || c > 'z') && (c < '0' || c > '9') && c != '-' && c != '_' {
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
		if (allowed.wildcard && strings.HasSuffix(host, allowed.host)) || host == allowed.host {
			return true
		}
	}
	return false
}
