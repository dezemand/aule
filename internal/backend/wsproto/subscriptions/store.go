package wssubscriptions

import (
	"sync"

	"github.com/dezemandje/aule/internal/backend/wsproto"
	"github.com/dezemandje/aule/internal/domain"
	"github.com/google/uuid"
)

type Subscription interface {
	ID() uuid.UUID
	ClientID() uuid.UUID
	UserID() domain.UserID
	Topic() string
	Query() any
}

type sub struct {
	id     uuid.UUID
	client *wsproto.Client
	userID domain.UserID
	topic  string
	query  any
}

func (s *sub) ID() uuid.UUID {
	return s.id
}

func (s *sub) ClientID() uuid.UUID {
	return s.client.ID()
}

func (s *sub) UserID() domain.UserID {
	return s.userID
}

func (s *sub) Topic() string {
	return s.topic
}

func (s *sub) Query() any {
	return s.query
}

func NewSubscription(client *wsproto.Client, topic string, query any) Subscription {
	return &sub{
		id:     uuid.New(),
		client: client,
		userID: client.UserID(),
		topic:  topic,
		query:  query,
	}
}

type Store interface {
	Add(sub Subscription) error

	Remove(clientID uuid.UUID, subscriptionID uuid.UUID) error
	RemoveAll(clientID uuid.UUID) error

	Find(string, func(Subscription) bool) ([]Subscription, error)
}

type memStore struct {
	mu   sync.RWMutex
	subs map[uuid.UUID]Subscription
}

func (m *memStore) Add(sub Subscription) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.subs[sub.ID()] = sub
	return nil
}

func (m *memStore) Find(topic string, cond func(Subscription) bool) ([]Subscription, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var results []Subscription
	for _, sub := range m.subs {
		if sub.Topic() == topic && cond(sub) {
			results = append(results, sub)
		}
	}

	return results, nil
}

func (m *memStore) Remove(clientID uuid.UUID, subscriptionID uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.subs, subscriptionID)
	return nil
}

func (m *memStore) RemoveAll(clientID uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for id, sub := range m.subs {
		if sub.ClientID() == clientID {
			delete(m.subs, id)
		}
	}
	return nil
}

func NewMemStore() Store {
	return &memStore{
		subs: make(map[uuid.UUID]Subscription),
	}
}
