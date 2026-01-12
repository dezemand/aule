package wsidempotency

import (
	"time"

	modelsws "github.com/dezemandje/aule/internal/model/ws"
	"github.com/google/uuid"
)

// CtxState represents the state of an idempotent request.
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
	messages *[]*modelsws.Envelope
	commitFn func()
}

func (c *idempotencyCtx) commit() {
	c.commitFn()
}
