package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/dezemandje/aule/internal/llmproxy/api"
	"github.com/dezemandje/aule/internal/llmproxy/config"
	"github.com/dezemandje/aule/internal/log"
	"github.com/joho/godotenv"
)

var logger log.Logger

func main() {
	// Load .env file if present
	_ = godotenv.Load()

	// Initialize logging
	log.Init()
	logger = log.GetLogger("cmd", "llmproxy")

	// Setup signal handling
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	// Load configuration
	cfg, err := config.NewConfigFromEnv()
	if err != nil {
		logger.Error("Failed to load config", "err", err)
		os.Exit(1)
	}

	// Create and start API
	server := api.New(cfg)

	// Handle shutdown in goroutine
	go func() {
		<-sigs
		logger.Info("Shutdown signal received")
		if err := server.Shutdown(); err != nil {
			logger.Error("Error during shutdown", "err", err)
		}
	}()

	// Start server (blocks until shutdown)
	if err := server.Start(); err != nil {
		logger.Error("Server error", "err", err)
		os.Exit(1)
	}

	logger.Info("LLM Proxy stopped")
}
