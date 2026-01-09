package wssubscriptions

import (
	"encoding/json"
	"errors"

	"github.com/dezemandje/aule/internal/backend/wsproto"
	"github.com/google/uuid"
)

var ErrSubscriptionTypeNotFound = errors.New("subscription type not found")

type Service struct {
	store Store
	types map[string]SubscriptionItem
}

func NewService(store Store) *Service {
	return &Service{
		store: store,
		types: make(map[string]SubscriptionItem),
	}
}

func (s *Service) Register(typ string, item SubscriptionItem) {
	s.types[typ] = item
}

func (s *Service) SendSubscriptionEvent(topic string, getSubs func(Subscription) bool, handler wsproto.HandlerFunc) error {
	subs, err := s.store.Find(topic, getSubs)
	if err != nil {
		return err
	}

	go func() {
		for _, sub := range subs {
			client, err := s.getClient(sub)
			if err != nil {
				continue
			}

			c := newSubCtx(client, sub.ID(), sub.Query())
			err = handler(c)
			if err != nil {
				continue
			}
		}
	}()

	return nil
}

func (s *Service) getClient(subscription Subscription) (*wsproto.Client, error) {
	sub, ok := subscription.(*sub)
	if !ok {
		return nil, ErrSubscriptionTypeNotFound
	}

	return sub.client, nil
}

func (s *Service) subscribe(client *wsproto.Client, topic string, query json.RawMessage, initial bool) (uuid.UUID, error) {
	typ, ok := s.types[topic]
	if !ok {
		return uuid.Nil, ErrSubscriptionTypeNotFound
	}

	subscription, err := typ.CreateSubscription(client, query)
	if err != nil {
		return uuid.Nil, err
	}

	if err := s.store.Add(subscription); err != nil {
		return uuid.Nil, err
	}

	if initial {
		err = typ.OnInitial(newSubCtx(client, subscription.ID(), subscription.Query()))
		if err != nil {
			// Who cares
		}
	}

	return subscription.ID(), nil
}

func (s *Service) unsubscribe(clientID uuid.UUID, subscriptionID uuid.UUID) error {
	if err := s.store.Remove(clientID, subscriptionID); err != nil {
		return err
	}

	return nil
}

func (s *Service) unsubscribeAll(clientID uuid.UUID) error {
	if err := s.store.RemoveAll(clientID); err != nil {
		return err
	}

	return nil
}
