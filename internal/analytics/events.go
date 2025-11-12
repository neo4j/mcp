package analytics

import (
	"fmt"
	"log"
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

type toolsProperties struct {
	baseProperties
	ToolUsed string `json:"tools_used"`
}

type TrackEvent struct {
	Event      string      `json:"event"`
	Properties interface{} `json:"properties"`
}

func (a *analytics) NewGDSProjCreatedEvent() TrackEvent {
	return TrackEvent{
		Event:      strings.Join([]string{eventNamePrefix, "GDS_PROJ_CREATED"}, "_"),
		Properties: getBaseProperties(a.cfg),
	}
}

func (a *analytics) NewGDSProjDropEvent() TrackEvent {
	return TrackEvent{
		Event:      strings.Join([]string{eventNamePrefix, "GDS_PROJ_DROP"}, "_"),
		Properties: getBaseProperties(a.cfg),
	}
}

func (a *analytics) NewStartupEvent() TrackEvent {
	return TrackEvent{
		Event:      strings.Join([]string{eventNamePrefix, "MCP_STARTUP"}, "_"),
		Properties: getBaseProperties(a.cfg),
	}
}

func (a *analytics) NewToolsEvent(toolsUsed string) TrackEvent {
	return TrackEvent{
		Event: strings.Join([]string{eventNamePrefix, "TOOL_USED"}, "_"),
		Properties: toolsProperties{
			baseProperties: getBaseProperties(a.cfg),
			ToolUsed:       toolsUsed,
		},
	}
}

func getBaseProperties(cfg analyticsConfig) baseProperties {
	uptime := time.Now().Unix() - cfg.startupTime
	insertID := newInsertID()
	return baseProperties{
		Token:      cfg.token,
		DistinctID: cfg.distinctID,
		Time:       time.Now().UnixMilli(),
		InsertID:   insertID,
		Uptime:     uptime,
		OS:         runtime.GOOS,
		OSArch:     runtime.GOARCH,
		IsAura:     cfg.isAura,
	}
}

func newInsertID() string {
	insertID, err := uuid.NewV6()
	if err != nil {
		insertIDerr := fmt.Errorf("error while generating uuid analytics events for analytics: %s", err.Error())
		log.Printf("MixPanel error: %s", insertIDerr.Error())
		return ""
	}
	return insertID.String()
}
