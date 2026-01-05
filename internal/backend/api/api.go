package api

import (
	"github.com/dezemandje/aule/internal/backend/api/userrest"
	"github.com/dezemandje/aule/internal/backend/auth"
	"github.com/dezemandje/aule/internal/backend/wsproto"
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
	authMw := auth.NewAuthMiddleware(ctx.AuthService)

	ctx.App.Route("/api", func(r fiber.Router) {
		r.Route("/auth", func(r fiber.Router) {
			authHandler := userrest.NewAuthHandler(ctx.AuthService)
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
	registerWsRoutes(ctx)
	ctx.UserWsHandler = wsproto.NewHandler(ctx.UserWsRouter)
}

func registerWsRoutes(ctx *ApiContext) {
	// ctx.UserWsRouter.Register("SomeMessageType", someHandlerFunction)
}
