package service

import (
	"context"

	"github.com/dezemandje/aule/internal/event"
	"github.com/dezemandje/aule/internal/eventhandler"
	"github.com/dezemandje/aule/internal/repository"
)

type ProjectService struct {
	eventHandler      eventhandler.EventHandler
	projectRepository repository.ProjectRepository
}

func NewProjectService() *ProjectService {
	return &ProjectService{}
}

func (s *ProjectService) CreateProject(ctx context.Context, name string, description string) error {
	id, err := s.projectRepository.Create(ctx, name, description)
	if err != nil {
		return err
	}

	evt := event.NewCreateProjectEvent(ctx, id, name)
	s.eventHandler.Emit(evt)

	return nil
}
