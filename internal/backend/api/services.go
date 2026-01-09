package api

import (
	"github.com/dezemandje/aule/internal/backend/auth"
	wssubscriptions "github.com/dezemandje/aule/internal/backend/wsproto/subscriptions"
	"github.com/dezemandje/aule/internal/eventhandler"
	projectsservice "github.com/dezemandje/aule/internal/service/project"
)

type Services struct {
	Events          eventhandler.EventHandler
	Auth            *auth.AuthService
	Project         *projectsservice.Service
	WsSubscriptions *wssubscriptions.Service
}

func setupServices(ctx *ApiContext) error {
	ctx.Services = &Services{}

	ctx.Services.WsSubscriptions = wssubscriptions.NewService(wssubscriptions.NewMemStore())
	ctx.Services.Events = nil
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

	return nil
}
