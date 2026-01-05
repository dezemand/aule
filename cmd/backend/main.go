package main

import (
	"time"

	"github.com/dezemandje/aule/internal/backend/api"
	"github.com/dezemandje/aule/internal/backend/config"
	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()

	providers := map[string]*config.OAuthConfig{
		"test": config.NewOAuthConfigFromEnv("OAUTH_test", "test"),
	}

	cfg := &config.Config{
		Auth: config.AuthConfig{
			JWTSecret:         "key",
			JWTExpiration:     time.Minute * 15,
			RefreshExpiration: time.Hour * 24 * 7,
			OAuthProviders:    providers,
		},
	}

	api := api.Setup(&api.ApiConfig{
		Config: cfg,
	})

	api.App.Listen(":9000")
}
