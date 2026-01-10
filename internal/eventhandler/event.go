package eventhandler

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type Metadata interface {
	ID() uuid.UUID
	TimeReceived() time.Time
}

type Event[T any] interface {
	Topic() Topic[T]
	Payload() T
	Context() context.Context
	Metadata() Metadata
}
