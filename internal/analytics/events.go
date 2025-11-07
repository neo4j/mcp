package analytics

import (
	"runtime"
	"strings"
	"time"
)

const eventNamePrefix = "MCP4NEO4J"

// TrackEventProperties are the properties attached to a MixPanel "track" event.
// DistinctID it's a distinct ID used to identify unique users, we do not use this information, therefore for us it will distinct different executions
// InsertID it's used to deduplicate duplicate messages.
type trackEventProperties struct {
	Token      string `json:"token"`
	Time       int64  `json:"time"`
	DistinctID string `json:"distinct_id"`
	InsertID   string `json:"$insert_id"`
}

type osInfoProperties struct {
	trackEventProperties
	OS     string `json:"os"`
	OSArch string `json:"os_arch"`
	Aura   bool   `json:"aura"`
}

type toolsProperties struct {
	trackEventProperties
	ToolUsed string `json:"tools_used"`
}

type trackEvent struct {
	Event      string      `json:"event"`
	Properties interface{} `json:"properties"`
}

func newGDSProjCreatedEvent(insertID string) trackEvent {
	return trackEvent{
		Event: strings.Join([]string{eventNamePrefix, "GDS_PROJ_CREATED"}, "_"),
		Properties: trackEventProperties{
			Token:      acfg.token,
			DistinctID: acfg.distinctID,
			Time:       time.Now().UnixMilli(),
			InsertID:   insertID,
		},
	}
}

func newGDSProjDropEvent(insertID string) trackEvent {
	return trackEvent{
		Event: strings.Join([]string{eventNamePrefix, "GDS_PROJ_DROP"}, "_"),
		Properties: trackEventProperties{
			Token:      acfg.token,
			DistinctID: acfg.distinctID,
			Time:       time.Now().UnixMilli(),
			InsertID:   insertID,
		},
	}
}

func newStartupEvent(insertID string) trackEvent {
	return trackEvent{
		Event: strings.Join([]string{eventNamePrefix, "MCP_STARTUP"}, "_"),
		Properties: trackEventProperties{
			Token:      acfg.token,
			DistinctID: acfg.distinctID,
			Time:       time.Now().UnixMilli(),
			InsertID:   insertID,
		},
	}
}

func newOSInfoEvent(insertID string, dbURI string) trackEvent {
	return trackEvent{
		Event: strings.Join([]string{eventNamePrefix, "OS_INFO"}, "_"),
		Properties: osInfoProperties{
			trackEventProperties: trackEventProperties{
				Token:      acfg.token,
				DistinctID: acfg.distinctID,
				Time:       time.Now().UnixMilli(),
				InsertID:   insertID,
			},
			OS:     runtime.GOOS,
			OSArch: runtime.GOARCH,
			Aura:   strings.Contains(dbURI, "database.neoj4.io"),
		},
	}
}

func newToolsEvent(insertID string, toolsUsed string) trackEvent {
	return trackEvent{
		Event: strings.Join([]string{eventNamePrefix, "TOOL_USED"}, "_"),
		Properties: toolsProperties{
			trackEventProperties: trackEventProperties{
				Token:      acfg.token,
				DistinctID: acfg.distinctID,
				Time:       time.Now().UnixMilli(),
				InsertID:   insertID,
			},
			ToolUsed: toolsUsed,
		},
	}
}
