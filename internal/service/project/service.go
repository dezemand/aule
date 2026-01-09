package projectsservice

import (
	"context"

	"github.com/dezemandje/aule/internal/event"
	"github.com/dezemandje/aule/internal/eventhandler"
)

type Service struct {
	eventHandler eventhandler.EventHandler
	repository   Repository
}

func NewService(eventHandler eventhandler.EventHandler, projectRepository Repository) *Service {
	return &Service{
		eventHandler: eventHandler,
		repository:   projectRepository,
	}
}

func (s *Service) CreateProject(ctx context.Context, name string, description string) error {
	id, err := s.repository.Create(ctx, name, description)
	if err != nil {
		return err
	}

	evt := event.NewCreateProjectEvent(ctx, id, name)
	s.eventHandler.Emit(evt)

	return nil
}
