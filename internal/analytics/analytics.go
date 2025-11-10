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
	client           HTTPClient
}

type Analytics struct {
	disabled bool
	acfg     AnalyticsConfig
}

// for testing purposes - enables dependency injection of http client
func NewAnalyticsWithClient(mixPanelToken string, mixpanelEndpoint string, client HTTPClient) Service {
	distinctID := getDistinctID()
	acfg := AnalyticsConfig{
		token:            mixPanelToken,
		mixpanelEndpoint: mixpanelEndpoint,
		distinctID:       distinctID,
		startupTime:      time.Now().Unix(),
		client:           client,
	}

	return &Analytics{acfg: acfg, disabled: false}
}

func NewAnalytics(mixPanelToken string, mixpanelEndpoint string) Service {
	distinctID := getDistinctID()
	acfg := AnalyticsConfig{
		token:            mixPanelToken,
		mixpanelEndpoint: mixpanelEndpoint,
		distinctID:       distinctID,
		startupTime:      time.Now().Unix(),
		client:           http.DefaultClient,
	}

	return &Analytics{acfg: acfg, disabled: false}
}

func (a *Analytics) EmitEvent(event TrackEvent) {
	if a.disabled {
		return
	}
	trackEvents := []TrackEvent{
		event,
	}

	log.Printf("Sending %s event to Neo4j", event.Event)
	err := a.sendTrackEvent(trackEvents)
	if err != nil {
		sendErr := fmt.Errorf("error while sending analytics events for analytics purpose: %s", err.Error())
		log.Printf("Analytics error: %s", sendErr.Error())
	}
}
func (a *Analytics) Enable() {
	a.disabled = false
}

func (a *Analytics) Disable() {
	a.disabled = true
}

// Eventually we can use mixpanel SDK
func (a *Analytics) sendTrackEvent(events []TrackEvent) error {
	b, err := json.Marshal(events)
	if err != nil {
		return fmt.Errorf("error appear while marshalling track event: %w", err)
	}
	url := strings.TrimRight(a.acfg.mixpanelEndpoint, "/") + "/track"

	resp, err := a.acfg.client.Post(url, "application/json; charset=utf-8", bytes.NewBuffer(b))
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
		log.Printf("error while generating distinct id for analytics purpose: %s", err.Error())
		return ""
	}
	return distinctID.String()
}
