package analytics

import (
	"context"
	"log"

	"github.com/google/uuid"
	"github.com/mixpanel/mixpanel-go"
)

// Analytics is a lightweight wrapper around the Mixpanel client.
type Analytics struct {
	uniqueID string
	client   mixpanel.ApiClient
	ctx      context.Context
}

// New creates a new Analytics instance with the given Mixpanel project token.
func New(ctx context.Context, projectToken string) *Analytics {
	// An unique ID will be generated per execution of the MCP Server
	id := uuid.New()

	return &Analytics{
		uniqueID: id.String(),
		client:   *mixpanel.NewApiClient(projectToken),
		ctx:      ctx,
	}
}

// TrackEvent sends an event to Mixpanel for a given distinctID with event name and properties
func (a *Analytics) TrackEvent(event Event) error {
	eventName := "NEO4JMCP_" + event.EventName()
	eventProperties := event.Properties()

	log.Printf("[analytics] sending event %s with id %s", eventName, a.uniqueID)

	// Send the event back to
	if err := a.client.Track(a.ctx, []*mixpanel.Event{a.client.NewEvent(eventName, a.uniqueID, eventProperties)}); err != nil {
		log.Printf("[analytics] failed to track event '%s': %v", eventName, err)
		return err
	}

	return nil
}
