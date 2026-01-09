package eventhandler

import (
	"context"

	"github.com/google/uuid"
)

type EventID uuid.UUID

type Event interface {
	ID() EventID
	Type() string
}

type EventBase struct {
	id EventID
}

func NewEventBase(ctx context.Context) EventBase {
	return EventBase{
		id: EventID(uuid.New()),
	}
}

func (e *EventBase) ID() EventID {
	return e.id
}

type Handler interface {
	Handle(Event) error
}

type EventHandler interface {
	Emit(Event) error
	Register(eventType string, handler Handler)
}
