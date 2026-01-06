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

func NewConfigFromEnv() Config {
	return Config{
		Auth:   NewAuthConfigFromEnv(),
		Server: NewServerConfigFromEnv(),
	}
}

func NewServerConfigFromEnv() ServerConfig {
	return ServerConfig{
		Host:         getEnv("SERVER_HOST", ""),
		Port:         getEnv("SERVER_PORT", "8080"),
		ReadTimeout:  getEnvDuration("SERVER_READ_TIMEOUT", 15*time.Second),
		WriteTimeout: getEnvDuration("SERVER_WRITE_TIMEOUT", 15*time.Second),
		IdleTimeout:  getEnvDuration("SERVER_IDLE_TIMEOUT", 60*time.Second),
	}
}
