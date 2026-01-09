package wsidempotency

import (
	"time"

	"github.com/dezemandje/aule/internal/backend/wsproto"
	"github.com/google/uuid"
)

type IdempotencyEntry struct {
	Created  time.Time           `json:"created"`
	State    CtxState            `json:"state"`
	Messages []*wsproto.Envelope `json:"messages"`
}

// IdempotencyStore is an interface for storing idempotency state.
type IdempotencyStore interface {
	Get(clientID uuid.UUID, key string, after time.Time) (*IdempotencyEntry, bool)
	Set(clientID uuid.UUID, key string, entry *IdempotencyEntry)
	Delete(clientID uuid.UUID, key string)
}

type memoryStore struct {
	store map[string]*IdempotencyEntry
}

func memKey(clientID uuid.UUID, key string) string {
	return clientID.String() + ":" + key
}

func (m *memoryStore) Delete(clientID uuid.UUID, key string) {
	delete(m.store, memKey(clientID, key))
}

func (m *memoryStore) Get(clientID uuid.UUID, key string, after time.Time) (*IdempotencyEntry, bool) {
	entry, ok := m.store[memKey(clientID, key)]
	return entry, ok
}

func (m *memoryStore) Set(clientID uuid.UUID, key string, entry *IdempotencyEntry) {
	m.store[memKey(clientID, key)] = entry
}

func NewMemoryStore() IdempotencyStore {
	return &memoryStore{
		store: make(map[string]*IdempotencyEntry),
	}
}
