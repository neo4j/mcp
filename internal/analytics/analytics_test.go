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
	"go.uber.org/mock/gomock"
)

func TestAnalytics(t *testing.T) {
	t.Run("EmitEvent should not send event if disabled", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		mockClient := amocks.NewMockHTTPClient(ctrl)

		analyticsService := analytics.NewAnalyticsWithClient("test-token", "http://localhost", mockClient, false)
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

		analyticsService := analytics.NewAnalyticsWithClient("test-token", "http://localhost", mockClient, false)
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

		analyticsService := analytics.NewAnalyticsWithClient("test-token", "http://localhost", mockClient, false)
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

		analyticsService := analytics.NewAnalyticsWithClient("test-token", "http://localhost", mockClient, false)
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

				analyticsService := analytics.NewAnalyticsWithClient("test-token", tc.mixpanelEndpoint, mockClient, false)
				analyticsService.EmitEvent(analytics.TrackEvent{Event: "test_event"})
			})
		}
	})
}

func TestEventCreation(t *testing.T) {
	analyticsService := analytics.NewAnalyticsWithClient("test-token", "http://localhost", nil, false)

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

	t.Run("NewToolsEvent", func(t *testing.T) {
		event := analyticsService.NewToolsEvent("gds")
		if event.Event != "MCP4NEO4J_TOOL_USED" {
			t.Errorf("unexpected event name: got %s, want %s", event.Event, "MCP4NEO4J_TOOL_USED")
		}
		props := assertBaseProperties(t, event.Properties)
		if props["tools_used"] != "gds" {
			t.Errorf("unexpected tools_used: got %v, want %v", props["tools_used"], "gds")
		}
	})

	t.Run("NewStartupEvent", func(t *testing.T) {
		event := analyticsService.NewStartupEvent()
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
	})

	t.Run("NewStartupEvent with Aura database", func(t *testing.T) {
		auraAnalytics := analytics.NewAnalyticsWithClient("test-token", "http://localhost", nil, true)
		event := auraAnalytics.NewStartupEvent()

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
