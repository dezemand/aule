package api

import (
	"github.com/dezemandje/aule/internal/backend/auth"
	"github.com/dezemandje/aule/internal/backend/config"
	"github.com/dezemandje/aule/internal/backend/wsproto"
	"github.com/gofiber/fiber/v2"
)

type ApiConfig struct {
	Config *config.Config
}

type ApiContext struct {
	Config *config.Config

	AuthService *auth.AuthService

	UserWsHandler *wsproto.Handler
	UserWsRouter  *wsproto.Router

	App *fiber.App
}

func Setup(cfg *ApiConfig) *ApiContext {
	ctx := &ApiContext{
		Config: cfg.Config,
	}

	setupAuth(ctx)
	setupWsRouter(ctx)
	setupHttpRouter(ctx)

	return ctx
}

func setupAuth(ctx *ApiContext) {
	ctx.AuthService = auth.NewAuthService(
		&ctx.Config.Auth,
		ctx.Config.Auth.OAuthProviders,
		&tmpRTRepo{store: make(map[string]string)},
	)
}

type tmpRTRepo struct {
	store map[string]string
}

func (r *tmpRTRepo) Create(userID string, token string) error {
	r.store[token] = userID
	return nil
}
func (r *tmpRTRepo) Find(token string) (string, bool) {
	userID, ok := r.store[token]
	if !ok {
		return "", false
	}
	return userID, true
}
func (r *tmpRTRepo) Delete(token string) error {
	delete(r.store, token)
	return nil
}
