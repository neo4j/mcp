package analytics

// Package analytics abstracts analytics handling for the program.
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

type analyticsConfig struct {
	token            string
	mixpanelEndpoint string
	distinctID       string
	startupTime      int64
	client           HTTPClient
	isAura           bool
}

type analytics struct {
	disabled bool
	cfg      analyticsConfig
}

// for testing purposes - enables dependency injection of http client
func NewAnalyticsWithClient(mixPanelToken string, mixpanelEndpoint string, client HTTPClient, isAura bool) Service {
	distinctID := getDistinctID()
	cfg := analyticsConfig{
		token:            mixPanelToken,
		mixpanelEndpoint: mixpanelEndpoint,
		distinctID:       distinctID,
		startupTime:      time.Now().Unix(),
		client:           client,
		isAura:           isAura,
	}

	return &analytics{cfg: cfg, disabled: false}
}

func NewAnalytics(mixPanelToken string, mixpanelEndpoint string, isAura bool) Service {
	distinctID := getDistinctID()
	cfg := analyticsConfig{
		token:            mixPanelToken,
		mixpanelEndpoint: mixpanelEndpoint,
		distinctID:       distinctID,
		startupTime:      time.Now().Unix(),
		client:           http.DefaultClient,
		isAura:           isAura,
	}

	return &analytics{cfg: cfg, disabled: false}
}

func (a *analytics) EmitEvent(event TrackEvent) {
	if a.disabled {
		return
	}
	trackEvents := []TrackEvent{
		event,
	}

	log.Printf("Sending %s event to Neo4j", event.Event)
	err := a.sendTrackEvent(trackEvents)
	if err != nil {
		sendErr := fmt.Errorf("error while sending analytics events for analytics: %s", err.Error())
		log.Printf("analytics error: %s", sendErr.Error())
	}
}
func (a *analytics) Enable() {
	a.disabled = false
}

func (a *analytics) Disable() {
	a.disabled = true
}

// Eventually we can use mixpanel SDK
func (a *analytics) sendTrackEvent(events []TrackEvent) error {
	b, err := json.Marshal(events)
	if err != nil {
		return fmt.Errorf("error while marshalling track event: %w", err)
	}
	url := strings.TrimRight(a.cfg.mixpanelEndpoint, "/") + "/track"

	resp, err := a.cfg.client.Post(url, "application/json; charset=utf-8", bytes.NewBuffer(b))
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

func getDistinctID() string {
	distinctID, err := uuid.NewV6()
	if err != nil {
		log.Printf("error while generating distinct id for analytics: %s", err.Error())
		return ""
	}
	return distinctID.String()
}
