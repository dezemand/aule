package wsidempotency

import (
	"time"

	"github.com/dezemandje/aule/internal/backend/wsproto"
	modelsws "github.com/dezemandje/aule/internal/model/ws"
	"github.com/google/uuid"
)

// Service handles idempotent request processing.
type Service struct {
	store IdempotencyStore
}

// NewService creates a new idempotency service.
func NewService(store IdempotencyStore) *Service {
	return &Service{
		store: store,
	}
}

// CheckAndProcess checks if a request is idempotent and returns cached response if available.
// Returns (shouldProcess, cachedMessages).
// If shouldProcess is true, the caller should process the request.
// If shouldProcess is false, cachedMessages contains the cached response to replay.
func (s *Service) CheckAndProcess(client *wsproto.Client, envelope *modelsws.Envelope) (shouldProcess bool, cachedMessages []*modelsws.Envelope) {
	if envelope.IdempotencyKey == "" {
		return true, nil
	}

	key := envelope.IdempotencyKey
	clientID := client.ID()
	idemCtx := s.getContext(clientID, key)

	if idemCtx.state == CtxDone {
		return false, *idemCtx.messages
	} else if idemCtx.state == CtxOngoing {
		// Request is already being processed - caller should skip
		return false, nil
	}

	return true, nil
}

// Complete marks an idempotent request as completed with the given response messages.
func (s *Service) Complete(clientID uuid.UUID, key string, messages []*modelsws.Envelope) {
	s.store.Set(clientID, key, &IdempotencyEntry{
		Messages: messages,
		State:    CtxDone,
		Created:  time.Now(),
	})
}

func (s *Service) getContext(clientID uuid.UUID, key string) *idempotencyCtx {
	exp := time.Now().Add(-2 * time.Hour)
	entry, ok := s.store.Get(clientID, key, exp)
	if !ok {
		messages := []*modelsws.Envelope{}
		created := time.Now()

		ctx := &idempotencyCtx{
			clientID: clientID,
			key:      key,
			state:    CtxOngoing,
			messages: &messages,
			commitFn: func() {
				s.store.Set(clientID, key, &IdempotencyEntry{
					Messages: messages,
					State:    CtxDone,
					Created:  created,
				})
			},
		}

		s.store.Set(clientID, key, &IdempotencyEntry{
			Messages: messages,
			State:    CtxOngoing,
			Created:  created,
		})

		return ctx
	}

	return &idempotencyCtx{
		clientID: clientID,
		key:      key,
		state:    entry.State,
		messages: &entry.Messages,
		commitFn: func() {},
	}
}
