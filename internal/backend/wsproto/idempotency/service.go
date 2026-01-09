package wsidempotency

import (
	"time"

	"github.com/dezemandje/aule/internal/backend/wsproto"
	"github.com/google/uuid"
)

type Service struct {
	store IdempotencyStore
}

func NewService(store IdempotencyStore) *Service {
	return &Service{
		store: store,
	}
}

func (s *Service) Middleware() func(wsproto.Ctx) error {
	return func(c wsproto.Ctx) error {
		if c.Message().IdempotencyKey == "" {
			return c.Next()
		}

		key := c.Message().IdempotencyKey
		clientID := c.Client().ID()
		idemCtx := s.getContext(clientID, key)

		if idemCtx.state == CtxDone {
			for _, msg := range *idemCtx.messages {
				err := c.Client().Send(msg)
				if err != nil {
					break
				}
			}
		} else if idemCtx.state == CtxOngoing {
			return c.ReplyError(
				"error.idempotency.ongoing",
				"The same request is already being processed",
				nil,
			)
		}

		return c.Next()
	}
}

func (s *Service) getContext(clientID uuid.UUID, key string) *idempotencyCtx {
	exp := time.Now().Add(-2 * time.Hour)
	entry, ok := s.store.Get(clientID, key, exp)
	if !ok {
		messages := []*wsproto.Envelope{}
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
	} else {
		return &idempotencyCtx{
			clientID: clientID,
			key:      key,
			state:    entry.State,
			messages: &entry.Messages,
			commitFn: func() { /* no-op */ },
		}
	}
}
