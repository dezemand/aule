package config

import (
	"time"
)

type Config struct {
	Server ServerConfig
	Auth   AuthConfig
}

type ServerConfig struct {
	Host         string
	Port         string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
}

type AuthConfig struct {
	JWTSecret         string
	JWTExpiration     time.Duration
	RefreshExpiration time.Duration
	OAuthProviders    map[string]*OAuthConfig
}
