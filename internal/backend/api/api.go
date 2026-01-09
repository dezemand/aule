package api

import (
	"github.com/dezemandje/aule/internal/backend/auth"
	"github.com/dezemandje/aule/internal/backend/wsproto"
	wsidempotency "github.com/dezemandje/aule/internal/backend/wsproto/idempotency"
	wssubscriptions "github.com/dezemandje/aule/internal/backend/wsproto/subscriptions"
	projectsservice "github.com/dezemandje/aule/internal/service/project"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/healthcheck"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/requestid"
)

func setupHttpRouter(ctx *ApiContext) {
	ctx.App = fiber.New()
	ctx.App.Use(requestid.New())
	ctx.App.Use(healthcheck.New())
	ctx.App.Use(logger.New())
	ctx.App.Use(func(c *fiber.Ctx) error {
		c.Locals("apicontext", ctx)
		return c.Next()
	})

	registerHttpRoutes(ctx)
}

func registerHttpRoutes(ctx *ApiContext) {
	authMw := auth.NewAuthMiddleware(ctx.Services.Auth)

	ctx.App.Route("/api", func(r fiber.Router) {
		r.Route("/auth", func(r fiber.Router) {
			authHandler := auth.NewAuthHandler(ctx.Services.Auth)
			r.Get("/", authHandler.GetJWT)
			r.Delete("/", authHandler.Logout)
			r.Get("/:provider/start", authHandler.StartOAuth)
			r.Get("/:provider/callback", authHandler.CallbackOAuth)
			r.Get("/providers", authHandler.GetProviders)
		})

		r.Get("/ws", authMw.RequireUserQuery, ctx.UserWsHandler.Handle)

		// app.Use(authMw.RequireUser)
		r.Get("/*", notFound)
	})

	ctx.App.Route("/agent", func(r fiber.Router) {
		// r.Use(authMw.RequireAgent)
		r.Get("/*", notFound)
	})

	ctx.App.Get("/*", notFound)
}

func notFound(c *fiber.Ctx) error {
	return fiber.ErrNotFound
}

func setupWsRouter(ctx *ApiContext) {
	ctx.UserWsRouter = wsproto.NewRouter()
	ctx.UserWsHandler = wsproto.NewHandler(ctx.UserWsRouter)
	registerWsRoutes(ctx)
}

type Hey struct {
	Message string `json:"message"`
}

func (h *Hey) Type() string {
	return "hey"
}

func registerWsRoutes(ctx *ApiContext) {
	store := wsidempotency.NewMemoryStore()
	idemMgr := wsidempotency.NewService(store)
	ctx.UserWsRouter.Use(idemMgr.Middleware())

	subscribeHandler := wssubscriptions.NewHandler(ctx.Services.WsSubscriptions)
	ctx.UserWsRouter.OnDisconnect(subscribeHandler.OnClose)
	ctx.UserWsRouter.On(wssubscriptions.MsgTypeSubscribe, subscribeHandler.OnSubscribe)
	ctx.UserWsRouter.On(wssubscriptions.MsgTypeUnsubscribe, subscribeHandler.OnUnsubscribe)

	projectsHander := projectsservice.NewWsHandler(ctx.Services.Events, ctx.Services.WsSubscriptions, ctx.Services.Project)
	ctx.UserWsRouter.On(projectsservice.MsgTypeProjectsList, projectsHander.OnListProjects)
	ctx.UserWsRouter.On(projectsservice.MsgTypeProjectCreate, projectsHander.OnCreateProject)

	ctx.UserWsRouter.OnConnect(func(c wsproto.Ctx) error {
		return c.Send(&Hey{Message: "Welcome to the Aule WebSocket API!"})
	})
}
