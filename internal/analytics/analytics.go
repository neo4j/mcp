package analytics

// Package analytics abstracts analytics handling for the program.
// Currently implemented for MixPanel.

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/neo4j/mcp/internal/logger"
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
	logger   *logger.Service
}

// for testing purposes - enables dependency injection of http client
func NewAnalyticsWithClient(mixPanelToken string, mixpanelEndpoint string, client HTTPClient, uri string, logger *logger.Service) (*Analytics, error) {
	if logger == nil {
		return nil, fmt.Errorf("logger cannot be nil")
	}

	distinctID := getDistinctID(logger)
	cfg := analyticsConfig{
		token:            mixPanelToken,
		mixpanelEndpoint: mixpanelEndpoint,
		distinctID:       distinctID,
		startupTime:      time.Now().Unix(),
		client:           client,
		isAura:           isAura(uri),
	}

	return &Analytics{cfg: cfg, disabled: false, logger: logger}, nil
}

func NewAnalytics(mixPanelToken string, mixpanelEndpoint string, uri string, logger *logger.Service) (*Analytics, error) {
	if logger == nil {
		return nil, fmt.Errorf("logger cannot be nil")
	}

	distinctID := getDistinctID(logger)
	cfg := analyticsConfig{
		token:            mixPanelToken,
		mixpanelEndpoint: mixpanelEndpoint,
		distinctID:       distinctID,
		startupTime:      time.Now().Unix(),
		client:           http.DefaultClient,
		isAura:           isAura(uri),
	}

	return &Analytics{cfg: cfg, disabled: false, logger: logger}, nil
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

	a.logger.Info("Sending event to Neo4j", "event", event.Event)
	err := a.sendTrackEvent(trackEvents)
	if err != nil {
		a.logger.Error("Error while sending analytics events", "error", err.Error())
	}
}
func (a *Analytics) Enable() {
	a.disabled = false
}

func (a *Analytics) Disable() {
	a.disabled = true
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
		a.logger.Error("Error while unmarshaling response from MixPanel", "error", err.Error())
	}

	a.logger.Info("Response from Neo4j", "status", resp.Status, "body", string(bodyBytes), "data", data)
	return nil
}

func getDistinctID(logger *logger.Service) string {
	distinctID, err := uuid.NewV6()
	if err != nil {
		logger.Error("Error while generating distinct ID for analytics", "error", err.Error())
		return ""
	}
	return distinctID.String()
}
