package analytics_test

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/neo4j/mcp/internal/analytics"
	"github.com/stretchr/testify/assert"
)

type MockHTTPClient struct {
	PostFunc func(url, contentType string, body io.Reader) (*http.Response, error)
}

func (m *MockHTTPClient) Post(url, contentType string, body io.Reader) (*http.Response, error) {
	if m.PostFunc != nil {
		return m.PostFunc(url, contentType, body)
	}
	return nil, nil
}

func TestAnalytics(t *testing.T) {
	t.Run("EmitEvent should send event if enabled", func(t *testing.T) {
		called := false
		mockClient := &MockHTTPClient{
			PostFunc: func(url, contentType string, body io.Reader) (*http.Response, error) {
				called = true
				assert.Equal(t, "http://localhost/track", url)
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader("1")),
				}, nil
			},
		}

		err := analytics.InitAnalyticsWithClient("test_token", "http://localhost", mockClient)
		assert.NoError(t, err)

		analytics.EmitEvent(analytics.TrackEvent{Event: "test_event"})
		assert.True(t, called)
	})
}
