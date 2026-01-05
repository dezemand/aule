package userws

import (
	"github.com/dezemandje/aule/internal/event"
	"github.com/dezemandje/aule/internal/eventhandler"
	"github.com/dezemandje/aule/internal/service"
)

type ProjectsHandler struct {
	projectsService service.ProjectService
	eventHandler    eventhandler.EventHandler
}

func NewProjectsHandler(projectsService service.ProjectService, eventHandler eventhandler.EventHandler) *ProjectsHandler {
	return &ProjectsHandler{
		projectsService: projectsService,
		eventHandler:    eventHandler,
	}
}

func (h *ProjectsHandler) onCreateProject(evt event.CreateProjectEvent) error {
	return nil
}
