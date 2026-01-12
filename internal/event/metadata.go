package event

import (
	"time"

	"github.com/google/uuid"
)

// EventMetadata provides concrete implementation of the Metadata interface.
// It carries identification and timing information for event tracing and ordering.
type EventMetadata struct {
	id            uuid.UUID
	timestamp     time.Time
	correlationID uuid.UUID // Links related events (e.g., request -> response)
	causationID   uuid.UUID // ID of the event that caused this one
}

// NewMetadata creates a new EventMetadata with a generated ID and current timestamp.
func NewMetadata() *EventMetadata {
	return &EventMetadata{
		id:        uuid.New(),
		timestamp: time.Now(),
	}
}

// NewMetadataWithCorrelation creates metadata with correlation tracking.
// Use this when an event is part of a larger workflow or transaction.
func NewMetadataWithCorrelation(correlationID, causationID uuid.UUID) *EventMetadata {
	return &EventMetadata{
		id:            uuid.New(),
		timestamp:     time.Now(),
		correlationID: correlationID,
		causationID:   causationID,
	}
}

// ID returns the unique identifier for this event.
func (m *EventMetadata) ID() uuid.UUID {
	return m.id
}

// Timestamp returns when this event was created.
func (m *EventMetadata) Timestamp() time.Time {
	return m.timestamp
}

// CorrelationID returns the correlation ID linking related events.
// Returns uuid.Nil if not set.
func (m *EventMetadata) CorrelationID() uuid.UUID {
	return m.correlationID
}

// CausationID returns the ID of the event that caused this one.
// Returns uuid.Nil if not set.
func (m *EventMetadata) CausationID() uuid.UUID {
	return m.causationID
}

// WithCorrelation returns a copy of the metadata with correlation IDs set.
func (m *EventMetadata) WithCorrelation(correlationID, causationID uuid.UUID) *EventMetadata {
	return &EventMetadata{
		id:            m.id,
		timestamp:     m.timestamp,
		correlationID: correlationID,
		causationID:   causationID,
	}
}
