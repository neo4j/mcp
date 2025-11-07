package analytics

import (
	"runtime"
	"strings"
	"time"
)

const eventNamePrefix = "MCP4NEO4J"

// baseProperties are the base properties attached to a MixPanel "track" event.
// DistinctID it's a distinct ID used to identify unique users, we do not use this information, therefore for us it will distinct different executions
// InsertID it's used to deduplicate duplicate messages.
type baseProperties struct {
	Token      string `json:"token"`
	Time       int64  `json:"time"`
	DistinctID string `json:"distinct_id"`
	InsertID   string `json:"$insert_id"`
	Uptime     int64  `json:"uptime"`
}

type osInfoProperties struct {
	baseProperties
	OS     string `json:"os"`
	OSArch string `json:"os_arch"`
	Aura   bool   `json:"aura"`
}

type toolsProperties struct {
	baseProperties
	ToolUsed string `json:"tools_used"`
}

type trackEvent struct {
	Event      string      `json:"event"`
	Properties interface{} `json:"properties"`
}

func newGDSProjCreatedEvent(insertID string) trackEvent {
	return trackEvent{
		Event:      strings.Join([]string{eventNamePrefix, "GDS_PROJ_CREATED"}, "_"),
		Properties: getBaseProperties(insertID),
	}
}

func newGDSProjDropEvent(insertID string) trackEvent {
	return trackEvent{
		Event:      strings.Join([]string{eventNamePrefix, "GDS_PROJ_DROP"}, "_"),
		Properties: getBaseProperties(insertID),
	}
}

func newStartupEvent(insertID string) trackEvent {
	return trackEvent{
		Event:      strings.Join([]string{eventNamePrefix, "MCP_STARTUP"}, "_"),
		Properties: getBaseProperties(insertID),
	}
}

func newOSInfoEvent(insertID string, dbURI string) trackEvent {
	return trackEvent{
		Event: strings.Join([]string{eventNamePrefix, "OS_INFO"}, "_"),
		Properties: osInfoProperties{
			baseProperties: getBaseProperties(insertID),
			OS:             runtime.GOOS,
			OSArch:         runtime.GOARCH,
			Aura:           strings.Contains(dbURI, "database.neoj4.io"),
		},
	}
}

func newToolsEvent(insertID string, toolsUsed string) trackEvent {
	return trackEvent{
		Event: strings.Join([]string{eventNamePrefix, "TOOL_USED"}, "_"),
		Properties: toolsProperties{
			baseProperties: getBaseProperties(insertID),
			ToolUsed:       toolsUsed,
		},
	}
}

func getBaseProperties(insertID string) baseProperties {
	uptime := time.Now().Unix() - acfg.startupTime
	return baseProperties{
		Token:      acfg.token,
		DistinctID: acfg.distinctID,
		Time:       time.Now().UnixMilli(),
		InsertID:   insertID,
		Uptime:     uptime,
	}
}
