//
// Copyright (c) "Neo4j"
// Neo4j Sweden AB [http://neo4j.com]

package analytics

//go:generate mockgen -destination=mocks/mock_analytics.go -package=analytics_mocks -typed github.com/neo4j/mcp/internal/analytics Service,HTTPClient
import (
	"io"
	"net/http"

	"github.com/neo4j/mcp/internal/config"
)

// Service
type Service interface {
	Disable()
	Enable()
	IsEnabled() bool
	EmitEvent(event TrackEvent)
	NewGDSProjCreatedEvent() TrackEvent
	NewGDSProjDropEvent() TrackEvent
	NewStartupEvent(transportMode config.TransportMode, tlsEnabled bool, mcpServer string) TrackEvent
	NewConnectionInitializedEvent(connInfo ConnectionEventInfo) TrackEvent
	NewToolEvent(toolsUsed string, success bool) TrackEvent
}

// dummy http client interface for our testing purposes
type HTTPClient interface {
	Post(url, contentType string, body io.Reader) (*http.Response, error)
}
