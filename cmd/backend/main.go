package main

import (
	"github.com/dezemandje/aule/internal/backend/api"
	"github.com/dezemandje/aule/internal/backend/config"
	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()
	cfg := config.NewConfigFromEnv()

	api, err := api.Setup(&cfg)
	if err != nil {
		panic(err)
	}

	err = api.Start()
	if err != nil {
		panic(err)
	}
}
