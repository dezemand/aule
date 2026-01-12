package api

import (
	"github.com/dezemandje/aule/internal/backend/config"
	"github.com/dezemandje/aule/internal/backend/wsproto"
	"github.com/gofiber/fiber/v2"
)

type ApiContext struct {
	Config *config.Config

	Data     *Data
	Services *Services

	UserWsHandler *wsproto.Handler

	App *fiber.App
}

func Setup(cfg *config.Config) (ctx *ApiContext, err error) {
	ctx = &ApiContext{Config: cfg}

	if err := setupData(ctx); err != nil {
		return nil, err
	}
	if err := setupServices(ctx); err != nil {
		return nil, err
	}
	setupWsRouter(ctx)
	setupHttpRouter(ctx)

	return ctx, nil
}

func (ctx *ApiContext) Start() error {
	return ctx.App.Listen(ctx.Config.Server.Host + ":" + ctx.Config.Server.Port)
}
