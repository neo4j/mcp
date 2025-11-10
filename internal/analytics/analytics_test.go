package analytics_test

import (
	"encoding/json"
	"io"
	"net/http"
	"runtime"
	"strings"
	"testing"

	"github.com/neo4j/mcp/internal/analytics"
	analytics_mocks "github.com/neo4j/mcp/internal/analytics/mocks"
	"go.uber.org/mock/gomock"
)

func TestAnalytics(t *testing.T) {
	t.Run("EmitEvent should not send event if disabled", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		mockClient := analytics_mocks.NewMockHTTPClient(ctrl)

		analyticsService := analytics.NewAnalyticsWithClient("test-token", "http://localhost", mockClient)
		analyticsService.Disable()
		analyticsService.EmitEvent(analytics.TrackEvent{Event: "test_event"})
	})

	t.Run("EmitEvent should send event if enabled", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		mockClient := analytics_mocks.NewMockHTTPClient(ctrl)

		mockClient.EXPECT().Post(gomock.Any(), gomock.Any(), gomock.Any()).Return(&http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader("1")),
		}, nil)

		analyticsService := analytics.NewAnalyticsWithClient("test-token", "http://localhost", mockClient)
		analyticsService.EmitEvent(analytics.TrackEvent{Event: "test_event"})
	})

	t.Run("EmitEvent should send the correct event in the body", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		mockClient := analytics_mocks.NewMockHTTPClient(ctrl)

		event := analytics.TrackEvent{
			Event: "specific_event",
			Properties: map[string]interface{}{
				"key": "value",
			},
		}

		mockClient.EXPECT().Post("http://localhost/track", gomock.Any(), gomock.Any()).
			DoAndReturn(func(url, contentType string, body io.Reader) (*http.Response, error) {
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

		analyticsService := analytics.NewAnalyticsWithClient("test-token", "http://localhost", mockClient)
		analyticsService.EmitEvent(event)
	})
}

func TestEventCreation(t *testing.T) {
	analyticsService := analytics.NewAnalyticsWithClient("test-token", "http://localhost", nil)

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

	t.Run("NewStartupEvent", func(t *testing.T) {
		event := analyticsService.NewStartupEvent()
		if event.Event != "MCP4NEO4J_MCP_STARTUP" {
			t.Errorf("unexpected event name: got %s, want %s", event.Event, "MCP4NEO4J_MCP_STARTUP")
		}
		assertBaseProperties(t, event.Properties)
	})

	t.Run("NewOSInfoEvent", func(t *testing.T) {
		event := analyticsService.NewOSInfoEvent("neo4j+s://test.database.neo4j.io")
		if event.Event != "MCP4NEO4J_OS_INFO" {
			t.Errorf("unexpected event name: got %s, want %s", event.Event, "MCP4NEO4J_OS_INFO")
		}
		props := assertBaseProperties(t, event.Properties)
		if props["os"] != runtime.GOOS {
			t.Errorf("unexpected os: got %v, want %v", props["os"], runtime.GOOS)
		}
		if props["os_arch"] != runtime.GOARCH {
			t.Errorf("unexpected os_arch: got %v, want %v", props["os_arch"], runtime.GOARCH)
		}
		if props["aura"] != true {
			t.Errorf("unexpected aura: got %v, want %v", props["aura"], true)
		}
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
	return m
}
