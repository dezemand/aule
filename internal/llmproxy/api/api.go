package api

import (
	"github.com/dezemandje/aule/internal/llmproxy/config"
	"github.com/dezemandje/aule/internal/llmproxy/provider"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	fiberlogger "github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/requestid"
)

// API holds the Fiber app and dependencies
type API struct {
	App      *fiber.App
	Config   *config.Config
	Registry *provider.Registry
}

// New creates a new API instance
func New(cfg *config.Config) *API {
	// Create provider registry
	registry := provider.NewRegistry()

	// Register OpenAI provider
	openaiProvider := provider.NewOpenAIProvider(cfg.OpenAI)
	registry.Register(openaiProvider)

	// Create Fiber app
	app := fiber.New(fiber.Config{
		DisableStartupMessage: true,
		StreamRequestBody:     true,
	})

	api := &API{
		App:      app,
		Config:   cfg,
		Registry: registry,
	}

	api.setupMiddleware()
	api.setupRoutes()

	return api
}

func (a *API) setupMiddleware() {
	// Recovery middleware
	a.App.Use(recover.New())

	// Request ID
	a.App.Use(requestid.New())

	// Logger
	a.App.Use(fiberlogger.New(fiberlogger.Config{
		Format: "[${time}] ${status} - ${latency} ${method} ${path}\n",
	}))

	// CORS (for development)
	a.App.Use(cors.New())
}

func (a *API) setupRoutes() {
	handler := NewHandler(a.Registry, a.Config)

	// Public routes
	a.App.Get("/v1/health", handler.HealthHandler)

	// Protected routes (require JWT)
	v1 := a.App.Group("/v1")
	v1.Use(JWTMiddleware(&a.Config.Auth))
	v1.Use(LLMConfigMiddleware(
		a.Registry,
		"openai",
		a.Config.OpenAI.DefaultModel,
		a.Config.Limits.MaxTokensPerRequest,
	))

	v1.Post("/complete", handler.CompleteHandler)
	v1.Get("/models", handler.ModelsHandler)
}

// Start starts the API server
func (a *API) Start() error {
	addr := a.Config.Server.Address()
	logger.Info("Starting LLM Proxy", "address", addr)
	return a.App.Listen(addr)
}

// Shutdown gracefully shuts down the API server
func (a *API) Shutdown() error {
	return a.App.Shutdown()
}
