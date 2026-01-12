package wssubscriptions

import (
	"context"
	"encoding/json"
	"errors"
	"sync"

	"github.com/dezemandje/aule/internal/backend/wsproto"
	"github.com/dezemandje/aule/internal/domain"
	"github.com/dezemandje/aule/internal/event"
	eventsws "github.com/dezemandje/aule/internal/model/events/ws"
	"github.com/google/uuid"
)

var ErrSubscriptionTypeNotFound = errors.New("subscription type not found")

// NotifyFunc is called for each matching subscription to produce an outgoing event.
// It receives the subscription and should return the outgoing event to send, or nil to skip.
type NotifyFunc func(sub Subscription) *eventsws.OutgoingEvent

// Service manages WebSocket subscriptions.
type Service struct {
	bus   *event.Bus
	store Store
	types map[string]SubscriptionItem
	mu    sync.RWMutex
}

// NewService creates a new subscription service.
func NewService(bus *event.Bus, store Store) *Service {
	return &Service{
		bus:   bus,
		store: store,
		types: make(map[string]SubscriptionItem),
	}
}

// Bus returns the event bus.
func (s *Service) Bus() *event.Bus {
	return s.bus
}

// Register registers a subscription type.
func (s *Service) Register(typ string, item SubscriptionItem) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.types[typ] = item
}

// Subscribe creates a subscription for a client.
func (s *Service) Subscribe(client *wsproto.Client, topic string, query json.RawMessage) (Subscription, error) {
	s.mu.RLock()
	typ, ok := s.types[topic]
	s.mu.RUnlock()

	if !ok {
		return nil, ErrSubscriptionTypeNotFound
	}

	subscription, err := typ.CreateSubscription(client, query)
	if err != nil {
		return nil, err
	}

	if err := s.store.Add(subscription); err != nil {
		return nil, err
	}

	return subscription, nil
}

// SendInitial sends the initial data for a subscription via the event bus.
func (s *Service) SendInitial(subscription Subscription) error {
	s.mu.RLock()
	typ, ok := s.types[subscription.Topic()]
	s.mu.RUnlock()

	if !ok {
		return ErrSubscriptionTypeNotFound
	}

	outgoing := typ.OnInitial(subscription)
	if outgoing == nil {
		return nil
	}

	event.Publish(s.bus, eventsws.TopicOutgoing.Event(*outgoing))
	return nil
}

// Unsubscribe removes a subscription.
func (s *Service) Unsubscribe(clientID uuid.UUID, subscriptionID uuid.UUID) error {
	return s.store.Remove(clientID, subscriptionID)
}

// UnsubscribeAll removes all subscriptions for a client.
func (s *Service) UnsubscribeAll(clientID uuid.UUID) error {
	return s.store.RemoveAll(clientID)
}

// Notify sends to all matching subscribers via the event bus.
func (s *Service) Notify(topic string, filter func(Subscription) bool, handler NotifyFunc) error {
	subs, err := s.store.Find(topic, filter)
	if err != nil {
		return err
	}

	go func() {
		for _, sub := range subs {
			outgoing := handler(sub)
			if outgoing == nil {
				continue
			}
			event.Publish(s.bus, eventsws.TopicOutgoing.Event(*outgoing))
		}
	}()

	return nil
}

// NotifyUser sends to all subscriptions for a specific user.
func (s *Service) NotifyUser(topic string, userID domain.UserID, handler NotifyFunc) error {
	return s.Notify(topic, func(sub Subscription) bool {
		return sub.UserID() == userID
	}, handler)
}

// NotifyAll sends to all subscriptions on a topic.
func (s *Service) NotifyAll(topic string, handler NotifyFunc) error {
	return s.Notify(topic, func(sub Subscription) bool {
		return true
	}, handler)
}

// SubscribeToBus subscribes to events on the event bus and notifies matching WS subscriptions.
func SubscribeToBus[T any](
	s *Service,
	eventTopic event.Topic[T],
	wsTopic string,
	bridge func(ctx context.Context, e event.Event[T]) (filter func(Subscription) bool, handler NotifyFunc),
) event.Subscription {
	if s.bus == nil {
		return nil
	}

	return event.Subscribe(s.bus, eventTopic, func(ctx context.Context, e event.Event[T]) error {
		filter, handler := bridge(ctx, e)
		return s.Notify(wsTopic, filter, handler)
	})
}
