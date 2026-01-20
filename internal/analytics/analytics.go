package analytics

// Package analytics abstracts analytics handling for the program.
// Currently implemented for MixPanel.

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
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

type Analytics struct {
	disabled bool
	cfg      analyticsConfig
}

// for testing purposes - enables dependency injection of http client
func NewAnalyticsWithClient(mixPanelToken string, mixpanelEndpoint string, client HTTPClient, uri string) *Analytics {
	distinctID := GetDistinctID()
	cfg := analyticsConfig{
		token:            mixPanelToken,
		mixpanelEndpoint: mixpanelEndpoint,
		distinctID:       distinctID,
		startupTime:      time.Now().Unix(),
		client:           client,
		isAura:           isAura(uri),
	}

	return &Analytics{cfg: cfg, disabled: false}
}

func NewAnalytics(mixPanelToken string, mixpanelEndpoint string, uri string) *Analytics {
	distinctID := GetDistinctID()
	cfg := analyticsConfig{
		token:            mixPanelToken,
		mixpanelEndpoint: mixpanelEndpoint,
		distinctID:       distinctID,
		startupTime:      time.Now().Unix(),
		client:           http.DefaultClient,
		isAura:           isAura(uri),
	}

	return &Analytics{cfg: cfg, disabled: false}
}

func isAura(uri string) bool {
	return strings.Contains(uri, "databases.neo4j.io")
}

func (a *Analytics) EmitEvent(event TrackEvent) {
	if a.disabled {
		return
	}
	trackEvents := []TrackEvent{
		event,
	}

	slog.Info("Sending event to Neo4j", "event", event.Event)
	err := a.sendTrackEvent(trackEvents)
	if err != nil {
		slog.Error("Error while sending analytics events", "error", err.Error())
	}
}
func (a *Analytics) Enable() {
	a.disabled = false
}

func (a *Analytics) Disable() {
	a.disabled = true
}

func (a *Analytics) IsEnabled() bool {
	return !a.disabled
}

func (a *Analytics) sendTrackEvent(events []TrackEvent) error {
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
	err = json.Unmarshal(bodyBytes, &data)
	if err != nil {
		slog.Error("Error while unmarshaling response from MixPanel", "error", err.Error())
	}

	slog.Info("Response from Neo4j", "status", resp.Status, "body", string(bodyBytes), "data", data)
	return nil
}

func GetDistinctID() string {
	distinctID, err := uuid.NewV6()
	if err != nil {
		slog.Error("Error while generating distinct ID for analytics", "error", err.Error())
		return ""
	}
	return distinctID.String()
}
