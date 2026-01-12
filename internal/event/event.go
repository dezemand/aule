package event

import (
	"time"

	"github.com/google/uuid"
)

// Metadata provides identification and timing information for events.
type Metadata interface {
	ID() uuid.UUID
	Timestamp() time.Time
}

// Event represents a domain event with a typed payload.
type Event[T any] interface {
	Topic() Topic[T]
	Payload() T
	Metadata() Metadata
}

// BaseEvent is a concrete implementation of Event[T].
// It wraps a typed payload with topic and metadata information.
type BaseEvent[T any] struct {
	topic    Topic[T]
	payload  T
	metadata Metadata
}

type eventConfig struct {
	source   Event[any]
	metadata Metadata
}

type EventOption func(*eventConfig)

// NewEvent creates a new event with the given topic and payload.
// Metadata is automatically generated with a new ID and current timestamp.
func NewEvent[T any](topic Topic[T], payload T, options ...EventOption) *BaseEvent[T] {
	cfg := eventConfig{}
	for _, opt := range options {
		opt(&cfg)
	}

	return &BaseEvent[T]{
		topic:    topic,
		payload:  payload,
		metadata: NewMetadata(),
	}
}

// Topic returns the event's topic.
func (e *BaseEvent[T]) Topic() Topic[T] {
	return e.topic
}

// Payload returns the event's typed payload.
func (e *BaseEvent[T]) Payload() T {
	return e.payload
}

// Metadata returns the event's metadata.
func (e *BaseEvent[T]) Metadata() Metadata {
	return e.metadata
}

// TopicString returns the topic as a dot-delimited string.
// This is useful for routing and logging.
func (e *BaseEvent[T]) TopicString() string {
	return e.topic.String()
}

func WithSource[T any](sourceEvent Event[T]) EventOption {
	return func(ec *eventConfig) {
		ec.source = &BaseEvent[any]{
			topic:    Topic[any](sourceEvent.Topic().Parts()),
			payload:  any(sourceEvent.Payload()),
			metadata: sourceEvent.Metadata(),
		}
	}
}
