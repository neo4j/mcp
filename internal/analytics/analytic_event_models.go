package analytics

// Event interface defines the contract for all trackable events
type Event interface {
	EventName() string
	Properties() map[string]any
}

// Holds information for a read cypher tool execution
// Only interested in the start of the cypher and any error message
type ReadCypherEvent struct {
	Event           string
	cypherOperation string // This will be MATCH, MERGE etc..
	cypherError     string // Any error message
}

func (r ReadCypherEvent) EventName() string {
	return r.Event
}

func (r ReadCypherEvent) Properties() map[string]any {

	props := map[string]any{
		"cypherOperation": r.cypherOperation,
		"cypherError":     r.cypherError,
	}
	return props
}

// Holds information for the environment this is running in.
type EnvReportEvent struct {
	Event           string // the event name
	neo4jmcpVersion string // the version
	OS              string // the name of the OS
	OSArch          string // the os arch
	Aura            bool   // True If using Aura
}

func (o EnvReportEvent) EventName() string {
	return o.Event
}

func (o EnvReportEvent) Properties() map[string]any {
	props := map[string]any{
		"neo4jmcpVersion": o.neo4jmcpVersion,
		"os":              o.OS,
		"osArch":          o.OSArch,
		"aura":            o.Aura,
	}
	return props
}
