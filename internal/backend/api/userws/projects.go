package userws

import (
	"context"

	"github.com/dezemandje/aule/internal/backend/wsproto"
	"github.com/dezemandje/aule/internal/event"
	"github.com/dezemandje/aule/internal/eventhandler"
	"github.com/dezemandje/aule/internal/service"
)

type ProjectsHandler struct {
	eventHandler    eventhandler.EventHandler
	projectsService *service.ProjectService
}

func NewProjectsHandler(eventHandler eventhandler.EventHandler, projectsService *service.ProjectService) *ProjectsHandler {
	return &ProjectsHandler{
		eventHandler:    eventHandler,
		projectsService: projectsService,
	}
}

func (h *ProjectsHandler) OnCreateProject(ctx context.Context, evt *event.CreateProjectEvent) error {
	wsproto.GetClient(ctx).UserID()
	return nil
}
