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

type startupProperties struct {
	baseProperties
	Neo4jVersion  string   `json:"neo4j_version"`
	Edition       string   `json:"edition"`
	CypherVersion []string `json:"cypher_version"`
	McpVersion    string   `json:"mcp_version"`
	TransportMode string   `json:"transport_mode"`
}

type httpStartupProperties struct {
	startupProperties
	TLSEnabled bool `json:"tls_enabled"`
}

type toolsProperties struct {
	baseProperties
	ToolUsed string `json:"tools_used"`
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

type StartupEventInfo struct {
	Neo4jVersion  string
	Edition       string
	CypherVersion []string
	McpVersion    string
}

func (a *Analytics) NewStartupEvent(startupInfoEvent StartupEventInfo) TrackEvent {
	baseStartupProps := startupProperties{
		baseProperties: a.getBaseProperties(),
		Neo4jVersion:   startupInfoEvent.Neo4jVersion,
		Edition:        startupInfoEvent.Edition,
		CypherVersion:  startupInfoEvent.CypherVersion,
		McpVersion:     startupInfoEvent.McpVersion,
		TransportMode:  a.cfg.transportMode,
	}

	// For HTTP mode, include TLS-specific properties
	var properties interface{}
	if a.cfg.transportMode == "http" {
		properties = httpStartupProperties{
			startupProperties: baseStartupProps,
			TLSEnabled:        a.cfg.tlsEnabled,
		}
	} else {
		properties = baseStartupProps
	}

	return TrackEvent{
		Event:      strings.Join([]string{eventNamePrefix, "MCP_STARTUP"}, "_"),
		Properties: properties,
	}
}

func (a *Analytics) NewToolsEvent(toolsUsed string) TrackEvent {
	return TrackEvent{
		Event: strings.Join([]string{eventNamePrefix, "TOOL_USED"}, "_"),
		Properties: toolsProperties{
			baseProperties: a.getBaseProperties(),
			ToolUsed:       toolsUsed,
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
