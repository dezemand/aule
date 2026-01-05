package event

import (
	"context"

	"github.com/dezemandje/aule/internal/eventhandler"
)

type CreateProjectEvent struct {
	source *eventhandler.EventSource
	ID     string `json:"id"`
	Name   string `json:"name"`
}

func NewCreateProjectEvent(ctx context.Context, id string, name string) CreateProjectEvent {
	return CreateProjectEvent{
		source: eventhandler.GetSource(ctx),
		ID:     id,
		Name:   name,
	}
}

func (e CreateProjectEvent) Type() string {
	return "CreateProjectEvent"
}

func (e CreateProjectEvent) Source() *eventhandler.EventSource {
	return e.source
}
