package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/dezemandje/aule/internal/backend/api"
	"github.com/dezemandje/aule/internal/backend/config"
	"github.com/dezemandje/aule/internal/log"
	"github.com/joho/godotenv"
)

var logger log.Logger

func main() {
	_ = godotenv.Load()

	log.Init()
	logger = log.GetLogger("cmd", "agent")

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	ctx, cancel := context.WithCancel(context.Background())

	cfg, err := config.NewConfigFromEnv()
	if err != nil {
		logger.Error("Failed to load config", "err", err)
		os.Exit(1)
	}

	api, err := api.Setup(&cfg)
	if err != nil {
		logger.Error("Failed setting up application's context", "err", err)
		os.Exit(1)
	}

	go func() {
		<-sigs
		api.App.Shutdown()
		logger.Info("Shutdown signal received")
		cancel()
	}()

	api.Services.Events.Start()

	err = api.Start()
	if err != nil {
		logger.Error("Failed starting application", "err", err)
		os.Exit(1)
	}

	_ = ctx

	logger.Info("Done?")
}
