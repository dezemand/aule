package wsidempotency

import (
	"time"

	"github.com/dezemandje/aule/internal/backend/wsproto"
	"github.com/google/uuid"
)

type CtxState int

const (
	CtxNew CtxState = iota
	CtxOngoing
	CtxDone
)

type idempotencyCtx struct {
	clientID uuid.UUID
	key      string
	created  time.Time
	state    CtxState
	messages *[]*wsproto.Envelope

	commitFn func()
}

func (c *idempotencyCtx) commit() {
	c.commitFn()
}
