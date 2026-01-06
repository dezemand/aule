package api

import (
	"github.com/dezemandje/aule/internal/backend/auth"
	"github.com/dezemandje/aule/internal/backend/config"
	"github.com/dezemandje/aule/internal/backend/wsproto"
	dbmemory "github.com/dezemandje/aule/internal/repository/memory"
	"github.com/gofiber/fiber/v2"
)

type ApiContext struct {
	Config *config.Config

	AuthService *auth.AuthService

	UserWsHandler *wsproto.Handler
	UserWsRouter  *wsproto.Router

	App *fiber.App
}

func Setup(cfg *config.Config) (*ApiContext, error) {
	ctx := &ApiContext{Config: cfg}

	setupAuth(ctx)
	setupWsRouter(ctx)
	setupHttpRouter(ctx)

	return ctx, nil
}

func (ctx *ApiContext) Start() error {
	return ctx.App.Listen(ctx.Config.Server.Host + ":" + ctx.Config.Server.Port)
}

func setupAuth(ctx *ApiContext) {
	ctx.AuthService = auth.NewAuthService(
		&ctx.Config.Auth,
		&ctx.Config.Auth.OAuthProviders,
		dbmemory.NewMemoryRefreshTokenRepository(),
		dbmemory.NewMemoryUserRepository(),
	)
}
