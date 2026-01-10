package api

import (
	"os"

	"github.com/dezemandje/aule/internal/backend/auth"
	wssubscriptions "github.com/dezemandje/aule/internal/backend/wsproto/subscriptions"
	"github.com/dezemandje/aule/internal/eventhandler"
	dbmemory "github.com/dezemandje/aule/internal/repository/memory"
	"github.com/dezemandje/aule/internal/service/agentapi"
	projectsservice "github.com/dezemandje/aule/internal/service/project"
)

type Services struct {
	Events          eventhandler.EventHandler
	Auth            *auth.AuthService
	Project         *projectsservice.Service
	AgentAPI        *agentapi.Service
	WsSubscriptions *wssubscriptions.Service
}

func setupServices(ctx *ApiContext) error {
	ctx.Services = &Services{}

	ctx.Services.WsSubscriptions = wssubscriptions.NewService(wssubscriptions.NewMemStore())
	ctx.Services.Events = eventhandler.NewMemoryEventHandler()
	ctx.Services.Auth = auth.NewAuthService(
		&ctx.Config.Auth,
		&ctx.Config.Auth.OAuthProviders,
		ctx.Data.RefreshTokenRepository,
		ctx.Data.UserRepository,
	)
	ctx.Services.Project = projectsservice.NewService(
		ctx.Services.Events,
		ctx.Data.ProjectRepository,
	)

	// Agent API service (using in-memory repos for now)
	taskRepo := dbmemory.NewTaskRepository()
	agentRepo := dbmemory.NewAgentInstanceRepository()
	logRepo := dbmemory.NewAgentLogRepository()

	// Get default work directory
	workDir, _ := os.Getwd()

	ctx.Services.AgentAPI = agentapi.NewService(
		ctx.Services.Events,
		taskRepo,
		agentRepo,
		logRepo,
		workDir,
	)

	return nil
}
