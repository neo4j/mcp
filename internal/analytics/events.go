package analytics

import (
	"log/slog"
	"runtime"
	"strings"
	"time"

	"github.com/google/uuid"
)

const eventNamePrefix = "MCP4NEO4J"

// baseProperties are the base properties attached to a MixPanel "track" event.
// DistinctID is a distinct ID used to identify unique users, we do not use this information, therefore for us it will be distinct different executions.
// InsertID is used to deduplicate duplicate messages.
type baseProperties struct {
	Token      string `json:"token"`
	Time       int64  `json:"time"`
	DistinctID string `json:"distinct_id"`
	InsertID   string `json:"$insert_id"`
	Uptime     int64  `json:"uptime"`
	OS         string `json:"$os"`
	OSArch     string `json:"os_arch"`
	IsAura     bool   `json:"isAura"`
}

// serverStartupProperties contains server-level information available at startup (no DB query required)
type serverStartupProperties struct {
	baseProperties
	McpVersion    string `json:"mcp_version"`
	TransportMode string `json:"transport_mode"`
	TLSEnabled    *bool  `json:"tls_enabled,omitempty"` // Only for HTTP mode, pointer allows explicit false
}

// connectionInitializedProperties contains Neo4j-specific information (requires DB query)
type connectionInitializedProperties struct {
	baseProperties
	Neo4jVersion  string   `json:"neo4j_version"`
	Edition       string   `json:"edition"`
	CypherVersion []string `json:"cypher_version"`
}

type toolsProperties struct {
	baseProperties
	ToolUsed string `json:"tools_used"`
}

// toolsWithContextProperties is used for HTTP mode to include DB context with each tool call
type toolsWithContextProperties struct {
	baseProperties
	ToolUsed      string   `json:"tools_used"`
	Neo4jVersion  string   `json:"neo4j_version"`
	Edition       string   `json:"edition"`
	CypherVersion []string `json:"cypher_version"`
}

type TrackEvent struct {
	Event      string      `json:"event"`
	Properties interface{} `json:"properties"`
}

func (a *Analytics) NewGDSProjCreatedEvent() TrackEvent {
	return TrackEvent{
		Event:      strings.Join([]string{eventNamePrefix, "GDS_PROJ_CREATED"}, "_"),
		Properties: a.getBaseProperties(),
	}
}

func (a *Analytics) NewGDSProjDropEvent() TrackEvent {
	return TrackEvent{
		Event:      strings.Join([]string{eventNamePrefix, "GDS_PROJ_DROP"}, "_"),
		Properties: a.getBaseProperties(),
	}
}

// ConnectionEventInfo contains Neo4j connection information obtained from database queries
type ConnectionEventInfo struct {
	Neo4jVersion  string
	Edition       string
	CypherVersion []string
}

// NewStartupEvent creates a server startup event with information available immediately (no DB query)
func (a *Analytics) NewStartupEvent() TrackEvent {
	props := serverStartupProperties{
		baseProperties: a.getBaseProperties(),
		McpVersion:     a.cfg.mcpVersion,
		TransportMode:  a.cfg.transportMode,
	}

	// Only include TLS field for HTTP mode (omitted for STDIO via omitempty tag with nil pointer)
	if a.cfg.transportMode == "http" {
		tlsEnabled := a.cfg.tlsEnabled
		props.TLSEnabled = &tlsEnabled
	}

	return TrackEvent{
		Event:      strings.Join([]string{eventNamePrefix, "MCP_STARTUP"}, "_"),
		Properties: props,
	}
}

// NewConnectionInitializedEvent creates a connection initialized event with DB information (STDIO mode only)
func (a *Analytics) NewConnectionInitializedEvent(connInfo ConnectionEventInfo) TrackEvent {
	return TrackEvent{
		Event: strings.Join([]string{eventNamePrefix, "CONNECTION_INITIALIZED"}, "_"),
		Properties: connectionInitializedProperties{
			baseProperties: a.getBaseProperties(),
			Neo4jVersion:   connInfo.Neo4jVersion,
			Edition:        connInfo.Edition,
			CypherVersion:  connInfo.CypherVersion,
		},
	}
}

// NewToolsEvent creates a tool usage event (STDIO mode - without DB context)
func (a *Analytics) NewToolsEvent(toolsUsed string) TrackEvent {
	return TrackEvent{
		Event: strings.Join([]string{eventNamePrefix, "TOOL_USED"}, "_"),
		Properties: toolsProperties{
			baseProperties: a.getBaseProperties(),
			ToolUsed:       toolsUsed,
		},
	}
}

// NewToolEventWithContext creates a tool usage event with DB context (HTTP mode only)
func (a *Analytics) NewToolEventWithContext(toolsUsed string, connInfo ConnectionEventInfo) TrackEvent {
	return TrackEvent{
		Event: strings.Join([]string{eventNamePrefix, "TOOL_USED"}, "_"),
		Properties: toolsWithContextProperties{
			baseProperties: a.getBaseProperties(),
			ToolUsed:       toolsUsed,
			Neo4jVersion:   connInfo.Neo4jVersion,
			Edition:        connInfo.Edition,
			CypherVersion:  connInfo.CypherVersion,
		},
	}
}

func (a *Analytics) getBaseProperties() baseProperties {
	uptime := time.Now().Unix() - a.cfg.startupTime
	insertID := a.newInsertID()
	return baseProperties{
		Token:      a.cfg.token,
		DistinctID: a.cfg.distinctID,
		Time:       time.Now().UnixMilli(),
		InsertID:   insertID,
		Uptime:     uptime,
		OS:         runtime.GOOS,
		OSArch:     runtime.GOARCH,
		IsAura:     a.cfg.isAura,
	}
}

func (a *Analytics) newInsertID() string {
	insertID, err := uuid.NewV6()
	if err != nil {
		slog.Error("Error while generating insert ID for analytics", "error", err.Error())
		return ""
	}
	return insertID.String()
}
