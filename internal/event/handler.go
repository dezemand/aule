package event

import (
	"context"
	"sync"

	"github.com/google/uuid"
)

// Handler processes events. Implementations should be safe for concurrent use.
type Handler interface {
	// Handle processes an event. The context carries cancellation and deadline.
	// Returns an error if processing fails.
	Handle(ctx context.Context, event any) error
}

// HandlerFunc is a function type that implements Handler.
type HandlerFunc func(ctx context.Context, event any) error

// Handle implements the Handler interface.
func (f HandlerFunc) Handle(ctx context.Context, event any) error {
	return f(ctx, event)
}

// Subscription represents an active event subscription.
// It can be used to unsubscribe from further events.
type Subscription interface {
	// ID returns the unique identifier for this subscription.
	ID() uuid.UUID

	// Topic returns the topic string this subscription is listening to.
	Topic() string

	// Unsubscribe removes this subscription from the bus.
	// Safe to call multiple times.
	Unsubscribe()
}

// subscription is the internal implementation of Subscription.
type subscription struct {
	id          uuid.UUID
	topic       string
	handler     Handler
	unsubscribe func()
	once        sync.Once
}

// newSubscription creates a new subscription instance.
func newSubscription(topic string, handler Handler, unsubscribeFn func()) *subscription {
	return &subscription{
		id:          uuid.New(),
		topic:       topic,
		handler:     handler,
		unsubscribe: unsubscribeFn,
	}
}

// ID returns the subscription's unique identifier.
func (s *subscription) ID() uuid.UUID {
	return s.id
}

// Topic returns the topic this subscription listens to.
func (s *subscription) Topic() string {
	return s.topic
}

// Unsubscribe removes this subscription. Safe to call multiple times.
func (s *subscription) Unsubscribe() {
	s.once.Do(func() {
		if s.unsubscribe != nil {
			s.unsubscribe()
		}
	})
}
