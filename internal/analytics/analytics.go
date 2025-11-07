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

type HTTPClient interface {
	Post(url, contentType string, body io.Reader) (*http.Response, error)
}

type AnalyticsConfig struct {
	token            string
	mixpanelEndpoint string
	distinctID       string
	startupTime      int64
	client           HTTPClient
}

var acfg *AnalyticsConfig = &AnalyticsConfig{}

var disabled bool = true

// for testing purposes - enables dependency injection of http client
func InitAnalyticsWithClient(mixPanelToken string, mixpanelEndpoint string, client HTTPClient) error {
	disabled = false
	distinctID, err := uuid.NewV6()
	if err != nil {
		return fmt.Errorf("error while generating distinct id for analytics purpose: %s", err.Error())
	}
	acfg.token = mixPanelToken
	acfg.mixpanelEndpoint = mixpanelEndpoint
	acfg.distinctID = distinctID.String()
	acfg.startupTime = time.Now().Unix()
	acfg.client = client

	return nil
}

func InitAnalytics(mixPanelToken string, mixpanelEndpoint string) error {
	disabled = false
	distinctID, err := uuid.NewV6()
	if err != nil {
		return fmt.Errorf("error while generating distinct id for analytics purpose: %s", err.Error())
	}
	acfg.token = mixPanelToken
	acfg.mixpanelEndpoint = mixpanelEndpoint
	acfg.distinctID = distinctID.String()
	acfg.startupTime = time.Now().Unix()
	acfg.client = http.DefaultClient

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

	resp, err := acfg.client.Post(url, "application/json; charset=utf-8", bytes.NewBuffer(b))
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
