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

var disabled bool = true

// Configure the Analytics package with external information
// When InitAnalytics is invoked, telemetry is enabled
func InitAnalytics(mixPanelToken string, mixpanelEndpoint string) error {
	disabled = false
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
func EmitEvent(event TrackEvent) {
	if disabled {
		return
	}

	trackEvents := []TrackEvent{
		event,
	}

	log.Printf("Sending %s event to Neo4j", event.Event)
	err := sendTrackEvent(trackEvents)
	if err != nil {
		sendErr := fmt.Errorf("error while sending analytics events for analytics purpose: %s", err.Error())
		log.Printf("Analytics error: %s", sendErr.Error())
	}
}

// Eventually we can use mixpanel SDK
func sendTrackEvent(events []TrackEvent) error {
	b, err := json.Marshal(events)
	if err != nil {
		return fmt.Errorf("error appear while marshalling track event: %w", err)
	}
	url := strings.TrimRight(acfg.mixpanelEndpoint, "/") + "/track"

	resp, err := http.Post(url, "application/json; charset=utf-8", bytes.NewBuffer(b))
	if err != nil {
		return fmt.Errorf("error while emitting analytics to Neo4j: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response body: %w", err)
	}

	// try to decode numeric response, fallback to raw body logging
	var data int32
	_ = json.Unmarshal(bodyBytes, &data)

	log.Printf("Response from Neo4j, Status: %s, Body: %s, Data: %d", resp.Status, string(bodyBytes), data)
	return nil
}
