package analytics

import (
	"io"
	"net/http"
)

//go:generate mockgen -destination=mocks/mock_analytics.go -package=analytics_mocks -typed github.com/neo4j/mcp/internal/analytics Service,HTTPClient

// Service
type Service interface {
	Disable()
	EmitEvent(event TrackEvent)
	Enable()
	NewGDSProjCreatedEvent() TrackEvent
	NewGDSProjDropEvent() TrackEvent
	NewOSInfoEvent(dbURI string) TrackEvent
	NewStartupEvent() TrackEvent
	NewToolsEvent(toolsUsed string) TrackEvent
}

// dummy http client interface for our testing purpose
type HTTPClient interface {
	Post(url, contentType string, body io.Reader) (*http.Response, error)
}
