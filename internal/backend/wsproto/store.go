package wsproto

import (
	"sync"

	"github.com/google/uuid"
)

type ClientStore interface {
	AddClient(client *Client) error
	RemoveClient(id uuid.UUID) error

	GetClient(id uuid.UUID) (*Client, bool)
}

type mapStore struct {
	mu      *sync.RWMutex
	clients map[uuid.UUID]*Client
}

func (m *mapStore) AddClient(client *Client) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.clients[client.ID()] = client
	return nil
}

func (m *mapStore) GetClient(id uuid.UUID) (*Client, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	client, ok := m.clients[id]
	return client, ok
}

func (m *mapStore) RemoveClient(id uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.clients, id)
	return nil
}

func NewMapClientStore() ClientStore {
	return &mapStore{
		mu:      &sync.RWMutex{},
		clients: make(map[uuid.UUID]*Client),
	}
}
