package analytics_test

import (
	"encoding/json"
	"io"
	"net/http"
	"runtime"
	"strings"
	"testing"

	"github.com/neo4j/mcp/internal/analytics"
	amocks "github.com/neo4j/mcp/internal/analytics/mocks"
	"github.com/neo4j/mcp/internal/config"
	"go.uber.org/mock/gomock"
)

// newTestAnalytics creates an analytics service for testing
func newTestAnalytics(t *testing.T, token, endpoint string, client analytics.HTTPClient, uri string) *analytics.Analytics {
	t.Helper()
	return analytics.NewAnalyticsWithClient(token, endpoint, client, uri, false, "1.0.0")
}

func TestAnalytics(t *testing.T) {
	t.Run("EmitEvent should not send event if disabled", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		mockClient := amocks.NewMockHTTPClient(ctrl)

		analyticsService := newTestAnalytics(t, "test-token", "http://localhost", mockClient, "bolt://localhost:7687")
		analyticsService.Disable()
		analyticsService.EmitEvent(analytics.TrackEvent{Event: "test_event"})
	})

	t.Run("EmitEvent should send event if enabled", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		mockClient := amocks.NewMockHTTPClient(ctrl)

		mockClient.EXPECT().Post(gomock.Any(), gomock.Any(), gomock.Any()).Return(&http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader("1")),
		}, nil)

		analyticsService := newTestAnalytics(t, "test-token", "http://localhost", mockClient, "bolt://localhost:7687")
		analyticsService.EmitEvent(analytics.TrackEvent{Event: "test_event"})
	})

	t.Run("EmitEvent should send the correct event in the body", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		mockClient := amocks.NewMockHTTPClient(ctrl)

		event := analytics.TrackEvent{
			Event: "specific_event",
			Properties: map[string]interface{}{
				"key": "value",
			},
		}

		mockClient.EXPECT().Post("http://localhost/track", gomock.Any(), gomock.Any()).
			DoAndReturn(func(_, _ string, body io.Reader) (*http.Response, error) {
				bodyBytes, err := io.ReadAll(body)
				if err != nil {
					t.Fatalf("error reading body: %v", err)
				}

				var decodedEvents []analytics.TrackEvent
				err = json.Unmarshal(bodyBytes, &decodedEvents)
				if err != nil {
					t.Fatalf("error unmarshalling body: %v", err)
				}
				if len(decodedEvents) != 1 {
					t.Fatalf("expected 1 event, got %d", len(decodedEvents))
				}
				decodedEvent := decodedEvents[0]

				if decodedEvent.Event != "specific_event" {
					t.Errorf("expected event 'specific_event', got '%s'", decodedEvent.Event)
				}
				properties, ok := decodedEvent.Properties.(map[string]interface{})
				if !ok {
					t.Fatalf("properties is not a map[string]interface{}")
				}
				if properties["key"] != "value" {
					t.Errorf("expected properties['key'] to be 'value', got '%v'", properties["key"])
				}

				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader("1")),
				}, nil
			})

		analyticsService := newTestAnalytics(t, "test-token", "http://localhost", mockClient, "bolt://localhost:7687")
		analyticsService.EmitEvent(event)
	})

	t.Run("EmitEvent should send the correct event in the body", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		mockClient := amocks.NewMockHTTPClient(ctrl)

		event := analytics.TrackEvent{
			Event: "specific_event",
			Properties: map[string]interface{}{
				"key": "value",
			},
		}

		mockClient.EXPECT().Post("http://localhost/track", gomock.Any(), gomock.Any()).
			DoAndReturn(func(_, _ string, body io.Reader) (*http.Response, error) {
				bodyBytes, err := io.ReadAll(body)
				if err != nil {
					t.Fatalf("error reading body: %v", err)
				}

				var decodedEvents []analytics.TrackEvent
				err = json.Unmarshal(bodyBytes, &decodedEvents)
				if err != nil {
					t.Fatalf("error unmarshalling body: %v", err)
				}
				if len(decodedEvents) != 1 {
					t.Fatalf("expected 1 event, got %d", len(decodedEvents))
				}
				decodedEvent := decodedEvents[0]

				if decodedEvent.Event != "specific_event" {
					t.Errorf("expected event 'specific_event', got '%s'", decodedEvent.Event)
				}
				properties, ok := decodedEvent.Properties.(map[string]interface{})
				if !ok {
					t.Fatalf("properties is not a map[string]interface{}")
				}
				if properties["key"] != "value" {
					t.Errorf("expected properties['key'] to be 'value', got '%v'", properties["key"])
				}

				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader("1")),
				}, nil
			})

		analyticsService := newTestAnalytics(t, "test-token", "http://localhost", mockClient, "bolt://localhost:7687")
		analyticsService.EmitEvent(event)
	})

	t.Run("EmitEvent should construct the correct URL (only one '/' between host and path)", func(t *testing.T) {
		testCases := []struct {
			name             string
			mixpanelEndpoint string
			expectedURL      string
		}{
			{
				name:             "endpoint with trailing slash",
				mixpanelEndpoint: "http://localhost/",
				expectedURL:      "http://localhost/track",
			},
			{
				name:             "endpoint without trailing slash",
				mixpanelEndpoint: "http://localhost",
				expectedURL:      "http://localhost/track",
			},
			{
				name:             "endpoint with multiple trailing slashes",
				mixpanelEndpoint: "http://localhost//",
				expectedURL:      "http://localhost/track",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				ctrl := gomock.NewController(t)
				mockClient := amocks.NewMockHTTPClient(ctrl)

				mockClient.EXPECT().Post(tc.expectedURL, gomock.Any(), gomock.Any()).Return(&http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader("1")),
				}, nil)

				analyticsService := newTestAnalytics(t, "test-token", tc.mixpanelEndpoint, mockClient, "bolt://localhost:7687")
				analyticsService.EmitEvent(analytics.TrackEvent{Event: "test_event"})
			})
		}
	})
}

func TestEventCreation(t *testing.T) {
	analyticsService := newTestAnalytics(t, "test-token", "http://localhost", nil, "bolt://localhost:7687")

	t.Run("NewGDSProjCreatedEvent", func(t *testing.T) {
		event := analyticsService.NewGDSProjCreatedEvent()
		if event.Event != "MCP4NEO4J_GDS_PROJ_CREATED" {
			t.Errorf("unexpected event name: got %s, want %s", event.Event, "MCP4NEO4J_GDS_PROJ_CREATED")
		}
		assertBaseProperties(t, event.Properties)
	})

	t.Run("NewGDSProjDropEvent", func(t *testing.T) {
		event := analyticsService.NewGDSProjDropEvent()
		if event.Event != "MCP4NEO4J_GDS_PROJ_DROP" {
			t.Errorf("unexpected event name: got %s, want %s", event.Event, "MCP4NEO4J_GDS_PROJ_DROP")
		}
		assertBaseProperties(t, event.Properties)
	})

	t.Run("NewToolEvent", func(t *testing.T) {
		event := analyticsService.NewToolEvent("gds", true)
		if event.Event != "MCP4NEO4J_TOOL_USED" {
			t.Errorf("unexpected event name: got %s, want %s", event.Event, "MCP4NEO4J_TOOL_USED")
		}
		props := assertBaseProperties(t, event.Properties)
		if props["tools_used"] != "gds" {
			t.Errorf("unexpected tools_used: got %v, want %v", props["tools_used"], "gds")
		}
		if props["success"] != true {
			t.Errorf("unexpected success: got %v, want %v", props["success"], true)
		}
		// Note: Neo4j connection info (version, edition, cypher version) is sent separately in CONNECTION_INITIALIZED event
	})

	t.Run("NewStartupEvent", func(t *testing.T) {
		event := analyticsService.NewStartupEvent(config.TransportModeStdio)
		if event.Event != "MCP4NEO4J_MCP_STARTUP" {
			t.Errorf("unexpected event name: got %s, want %s", event.Event, "MCP4NEO4J_MCP_STARTUP")
		}
		props := assertBaseProperties(t, event.Properties)
		if props["$os"] != runtime.GOOS {
			t.Errorf("unexpected os: got %v, want %v", props["os"], runtime.GOOS)
		}
		if props["os_arch"] != runtime.GOARCH {
			t.Errorf("unexpected os_arch: got %v, want %v", props["os_arch"], runtime.GOARCH)
		}
		if props["isAura"] == true {
			t.Errorf("unexpected aura: got %v, want %v", props["isAura"], false)
		}
		if props["mcp_version"] != "1.0.0" {
			t.Errorf("unexpected mcp_version: got %v, want %v", props["mcp_version"], "1.0.0")
		}
		if props["transport_mode"] != "stdio" {
			t.Errorf("unexpected transport_mode: got %v, want %v", props["transport_mode"], "stdio")
		}
	})

	t.Run("NewConnectionInitializedEvent", func(t *testing.T) {
		event := analyticsService.NewConnectionInitializedEvent(analytics.ConnectionEventInfo{
			Neo4jVersion:  "2025.09.01",
			CypherVersion: []string{"5", "25"},
			Edition:       "enterprise",
		})
		if event.Event != "MCP4NEO4J_CONNECTION_INITIALIZED" {
			t.Errorf("unexpected event name: got %s, want %s", event.Event, "MCP4NEO4J_CONNECTION_INITIALIZED")
		}
		props := assertBaseProperties(t, event.Properties)
		if props["neo4j_version"] != "2025.09.01" {
			t.Errorf("unexpected Neo4jVersion: got %v, want %v", props["neo4j_version"], "2025.09.01")
		}
		if props["edition"] != "enterprise" {
			t.Errorf("unexpected edition: got %v, want %v", props["edition"], "enterprise")
		}

		cypherVersion, ok := props["cypher_version"].([]interface{})
		if !ok {
			t.Fatalf("cypher_version is not a []interface{}")
		}
		if len(cypherVersion) != 2 || cypherVersion[0] != "5" || cypherVersion[1] != "25" {
			t.Errorf("unexpected cypher_version: got %v, want %v", props["cypher_version"], []string{"5", "25"})
		}
	})

	t.Run("NewStartupEvent with Aura database", func(t *testing.T) {
		auraAnalytics := newTestAnalytics(t, "test-token", "http://localhost", nil, "bolt://mydb.databases.neo4j.io")
		event := auraAnalytics.NewStartupEvent(config.TransportModeHTTP)

		if event.Event != "MCP4NEO4J_MCP_STARTUP" {
			t.Errorf("unexpected event name: got %s, want %s", event.Event, "MCP4NEO4J_MCP_STARTUP")
		}
		props := assertBaseProperties(t, event.Properties)
		if props["$os"] != runtime.GOOS {
			t.Errorf("unexpected os: got %v, want %v", props["os"], runtime.GOOS)
		}
		if props["os_arch"] != runtime.GOARCH {
			t.Errorf("unexpected os_arch: got %v, want %v", props["os_arch"], runtime.GOARCH)
		}
		if props["isAura"] == false {
			t.Errorf("unexpected aura: got %v, want %v", props["isAura"], true)
		}
		if props["mcp_version"] != "1.0.0" {
			t.Errorf("unexpected mcp_version: got %v, want %v", props["mcp_version"], "1.0.0")
		}
	})

	t.Run("NewStartupEvent with STDIO transport mode", func(t *testing.T) {
		stdioAnalytics := analytics.NewAnalyticsWithClient(
			"test-token",
			"http://localhost",
			nil,
			"bolt://localhost:7687",
			false,
			"1.0.0",
		)
		event := stdioAnalytics.NewStartupEvent(config.TransportModeStdio)

		if event.Event != "MCP4NEO4J_MCP_STARTUP" {
			t.Errorf("unexpected event name: got %s, want %s", event.Event, "MCP4NEO4J_MCP_STARTUP")
		}

		props := assertBaseProperties(t, event.Properties)

		// Verify transport_mode is set to "stdio"
		if props["transport_mode"] != "stdio" {
			t.Errorf("unexpected transport_mode: got %v, want %v", props["transport_mode"], "stdio")
		}

		// Verify tls_enabled is NOT present in STDIO mode (uses omitempty)
		if _, exists := props["tls_enabled"]; exists {
			t.Errorf("tls_enabled should not be present in STDIO mode, but found: %v", props["tls_enabled"])
		}
	})

	t.Run("NewStartupEvent with HTTP transport mode and TLS enabled", func(t *testing.T) {
		httpAnalytics := analytics.NewAnalyticsWithClient(
			"test-token",
			"http://localhost",
			nil,
			"bolt://localhost:7687",
			true,
			"1.0.0",
		)
		event := httpAnalytics.NewStartupEvent(config.TransportModeHTTP)

		if event.Event != "MCP4NEO4J_MCP_STARTUP" {
			t.Errorf("unexpected event name: got %s, want %s", event.Event, "MCP4NEO4J_MCP_STARTUP")
		}

		props := assertBaseProperties(t, event.Properties)

		// Verify transport_mode is set to "http"
		if props["transport_mode"] != "http" {
			t.Errorf("unexpected transport_mode: got %v, want %v", props["transport_mode"], "http")
		}

		// Verify tls_enabled is present and set to true in HTTP mode
		tlsEnabled, exists := props["tls_enabled"]
		if !exists {
			t.Errorf("tls_enabled should be present in HTTP mode")
		} else if tlsEnabled != true {
			t.Errorf("unexpected tls_enabled: got %v, want %v", tlsEnabled, true)
		}
	})

	t.Run("NewStartupEvent with HTTP transport mode and TLS disabled", func(t *testing.T) {
		httpAnalytics := analytics.NewAnalyticsWithClient(
			"test-token",
			"http://localhost",
			nil,
			"bolt://localhost:7687",
			false,
			"1.0.0",
		)
		event := httpAnalytics.NewStartupEvent(config.TransportModeHTTP)

		if event.Event != "MCP4NEO4J_MCP_STARTUP" {
			t.Errorf("unexpected event name: got %s, want %s", event.Event, "MCP4NEO4J_MCP_STARTUP")
		}

		props := assertBaseProperties(t, event.Properties)

		// Verify transport_mode is set to "http"
		if props["transport_mode"] != "http" {
			t.Errorf("unexpected transport_mode: got %v, want %v", props["transport_mode"], "http")
		}

		// Verify tls_enabled is present and set to false in HTTP mode
		tlsEnabled, exists := props["tls_enabled"]
		if !exists {
			t.Errorf("tls_enabled should be present in HTTP mode")
		} else if tlsEnabled != false {
			t.Errorf("unexpected tls_enabled: got %v, want %v", tlsEnabled, false)
		}
	})

}

func assertBaseProperties(t *testing.T, props interface{}) map[string]interface{} {
	t.Helper()
	p, err := json.Marshal(props)
	if err != nil {
		t.Fatalf("failed to marshal properties: %v", err)
	}
	var m map[string]interface{}
	if err := json.Unmarshal(p, &m); err != nil {
		t.Fatalf("failed to unmarshal properties to map: %v", err)
	}

	if m["token"] != "test-token" {
		t.Errorf("unexpected token: got %v, want %v", m["token"], "test-token")
	}
	if _, ok := m["time"].(float64); !ok {
		t.Errorf("time is not a number")
	}
	if _, ok := m["distinct_id"].(string); !ok {
		t.Errorf("distinct_id is not a string")
	}
	if _, ok := m["$insert_id"].(string); !ok {
		t.Errorf("$insert_id is not a string")
	}
	if _, ok := m["uptime"].(float64); !ok {
		t.Errorf("uptime is not a number")
	}
	if _, ok := m["$os"].(string); !ok {
		t.Errorf("$os is not a string")
	}
	if _, ok := m["os_arch"].(string); !ok {
		t.Errorf("os_arch is not a string")
	}
	if _, ok := m["isAura"].(bool); !ok {
		t.Errorf("isAura is not a bool")
	}
	return m
}
