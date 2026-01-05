package eventhandler

import "context"

type EventSource struct {
}

type Event interface {
	Type() string
	Source() *EventSource
}

type EventHandler interface {
	Emit(Event) error
}

func GetSource(ctx context.Context) *EventSource {
	return &EventSource{}
}
