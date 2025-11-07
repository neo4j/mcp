package analytics

// Package analytics abstracts analytics handling for the repository.
// Currently implemented for MixPanel.

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
)

type AnalyticsConfig struct {
	token            string
	mixpanelEndpoint string
	distinctID       string
	startupTime      int64
}

var acfg *AnalyticsConfig

// Configure the Analytics package with external information
func InitAnalytics(mixPanelToken string, mixpanelEndpoint string) error {
	distinctID, err := uuid.NewV6()
	if err != nil {
		return fmt.Errorf("error while generating distinct id for analytics purpose: %s", err.Error())
	}
	acfg = &AnalyticsConfig{
		token:            mixPanelToken,
		mixpanelEndpoint: mixpanelEndpoint,
		distinctID:       distinctID.String(),
		startupTime:      time.Now().Unix(),
	}

	return nil
}

func EmitStartupEvent() {
	if acfg == nil {
		return
	}
	insertID := newInsertID()
	trackEvents := []trackEvent{
		newStartupEvent(insertID),
	}
	err := sendTrackEvent(trackEvents)
	if err != nil {
		sendErr := fmt.Errorf("error while sending analytics events for analytics purpose: %s", err.Error())
		log.Printf("MixPanel error: %s", sendErr.Error())
	}
}

func EmitOSEvent(dbURI string) {
	if acfg == nil {
		return
	}
	insertID := newInsertID()
	trackEvents := []trackEvent{
		newOSInfoEvent(insertID, dbURI),
	}
	err := sendTrackEvent(trackEvents)
	if err != nil {
		sendErr := fmt.Errorf("error while sending analytics events for analytics purpose: %s", err.Error())
		log.Printf("MixPanel error: %s", sendErr.Error())
	}
}

func EmitToolUsedEvent(toolName string) {
	if acfg == nil {
		return
	}
	insertID := newInsertID()
	trackEvents := []trackEvent{
		newToolsEvent(insertID, toolName),
	}
	err := sendTrackEvent(trackEvents)
	if err != nil {
		sendErr := fmt.Errorf("error while sending analytics events for analytics purpose: %s", err.Error())
		log.Printf("MixPanel error: %s", sendErr.Error())
	}
}
func EmitGDSProjCreatedEvent() {
	if acfg == nil {
		return
	}
	insertID := newInsertID()
	trackEvents := []trackEvent{
		newGDSProjCreatedEvent(insertID),
	}
	err := sendTrackEvent(trackEvents)
	if err != nil {
		sendErr := fmt.Errorf("error while sending analytics events for analytics purpose: %s", err.Error())
		log.Printf("MixPanel error: %s", sendErr.Error())
	}
}

func EmitGDSProjDropEvent() {
	if acfg == nil {
		return
	}
	insertID := newInsertID()
	trackEvents := []trackEvent{
		newGDSProjDropEvent(insertID),
	}
	err := sendTrackEvent(trackEvents)
	if err != nil {
		sendErr := fmt.Errorf("error while sending analytics events for analytics purpose: %s", err.Error())
		log.Printf("MixPanel error: %s", sendErr.Error())
	}
}
func newInsertID() string {
	insertID, err := uuid.NewV6()
	if err != nil {
		insertIDerr := fmt.Errorf("error while sending analytics events for analytics purpose: %s", err.Error())
		log.Printf("MixPanel error: %s", insertIDerr.Error())
		return ""
	}
	return insertID.String()
}

// Eventually we can use mixpanel SDK
func sendTrackEvent(events []trackEvent) error {
	b, err := json.Marshal(events)
	if err != nil {
		return fmt.Errorf("error appear while marshalling track event: %w", err)
	}
	url := strings.TrimRight(acfg.mixpanelEndpoint, "/") + "/track"

	resp, err := http.Post(url, "application/json; charset=utf-8", bytes.NewBuffer(b))
	if err != nil {
		return fmt.Errorf("error while emitting analytics to MixPanel: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response body: %w", err)
	}

	// try to decode numeric response, fallback to raw body logging
	var data int32
	_ = json.Unmarshal(bodyBytes, &data)

	log.Printf("Response from MixPanel, Status: %s, Body: %s, Data: %d", resp.Status, string(bodyBytes), data)
	return nil
}
