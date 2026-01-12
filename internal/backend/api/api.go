package api

import (
	"github.com/dezemandje/aule/internal/backend/auth"
	"github.com/dezemandje/aule/internal/backend/wsproto"
	wssubscriptions "github.com/dezemandje/aule/internal/backend/wsproto/subscriptions"
	"github.com/dezemandje/aule/internal/service/agentapi"
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
		r.Use(authMw.RequireAgent)
		agentHandler := agentapi.NewHandler(ctx.Services.AgentAPI)

		r.Route("/v1/tasks", func(r fiber.Router) {
			r.Get("/:task_id", agentHandler.GetTask)
			r.Post("/:task_id/start", agentHandler.StartTask)
			r.Post("/:task_id/update", agentHandler.UpdateTask)
			r.Post("/:task_id/complete", agentHandler.CompleteTask)
			r.Post("/:task_id/fail", agentHandler.FailTask)
		})

		r.Get("/*", notFound)
	})

	ctx.App.Get("/*", notFound)
}

func notFound(c *fiber.Ctx) error {
	return fiber.ErrNotFound
}

func setupWsRouter(ctx *ApiContext) {
	ctx.UserWsHandler = wsproto.NewHandler(ctx.Services.Events, wsproto.NewMapClientStore())
	ctx.Handlers = &Handlers{}
	setupEventHandlers(ctx)
}

func setupEventHandlers(ctx *ApiContext) {
	// Subscription handler - handles subscribe/unsubscribe messages
	ctx.Handlers.Subscriptions = wssubscriptions.NewHandler(
		ctx.Services.Events,
		ctx.Services.WsSubscriptions,
		ctx.UserWsHandler,
	)
	ctx.Handlers.Subscriptions.SetupEventHandlers()

	// Projects handler - handles project CRUD via WS
	ctx.Handlers.Projects = projectsservice.NewHandler(
		ctx.Services.Events,
		ctx.Services.Project,
		ctx.Services.WsSubscriptions,
	)
	ctx.Handlers.Projects.SetupEventHandlers()
}
