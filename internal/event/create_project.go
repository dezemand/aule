package event

import (
	"context"

	"github.com/dezemandje/aule/internal/domain"
	"github.com/dezemandje/aule/internal/eventhandler"
)

type CreateProjectEvent struct {
	eventhandler.EventBase
	ProjectID domain.ProjectID `json:"project_id"`
	CreatorID domain.UserID    `json:"creator_id"`
	Name      string           `json:"name"`
}

func NewCreateProjectEvent(ctx context.Context, id domain.ProjectID, name string) *CreateProjectEvent {
	return &CreateProjectEvent{
		EventBase: eventhandler.NewEventBase(ctx),
		ProjectID: id,
		Name:      name,
	}
}

func (e *CreateProjectEvent) Type() string {
	return "create_project"
}
