package config

import (
	"time"

	"github.com/valyala/fasttemplate"
)

type AuthConfig struct {
	JWTSecret         string
	JWTExpiration     time.Duration
	RefreshExpiration time.Duration
	OAuthProviders    map[string]OAuthProviderConfig
}

func NewAuthConfigFromEnv() AuthConfig {
	providerIds := getEnvStringSlice("OAUTH_PROVIDERS", []string{})
	redirectUrl := getEnv("OAUTH_REDIRECT_URL", "http://localhost:8080/api/auth/oauth/callback")
	providers := make(map[string]OAuthProviderConfig)

	for _, id := range providerIds {
		cfg := NewOAuthConfigFromEnv("OAUTH_"+id, id)
		cfg.RedirectURL = fasttemplate.ExecuteString(redirectUrl, "{", "}", map[string]interface{}{
			"provider": id,
		})
		providers[id] = cfg
	}

	return AuthConfig{
		JWTSecret:         getEnv("AUTH_JWT_SECRET", ""),
		JWTExpiration:     getEnvDuration("AUTH_JWT_EXPIRATION", 15*time.Minute),
		RefreshExpiration: getEnvDuration("AUTH_REFRESH_EXPIRATION", 7*24*time.Hour),
		OAuthProviders:    providers,
	}
}
