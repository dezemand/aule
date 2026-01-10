package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config holds all configuration for the LLM Proxy
type Config struct {
	Server ServerConfig
	Auth   AuthConfig
	OpenAI OpenAIConfig
	Limits LimitsConfig
}

// ServerConfig holds HTTP server configuration
type ServerConfig struct {
	Host string
	Port string
}

// AuthConfig holds authentication configuration
type AuthConfig struct {
	JWTSecret string
}

// OpenAIConfig holds OpenAI provider configuration
type OpenAIConfig struct {
	APIKey       string
	BaseURL      string
	DefaultModel string
}

// LimitsConfig holds rate limiting and resource limits
type LimitsConfig struct {
	MaxTokensPerRequest int
	RequestTimeout      time.Duration
	StreamingTimeout    time.Duration
}

// NewConfigFromEnv loads configuration from environment variables
func NewConfigFromEnv() (*Config, error) {
	cfg := &Config{
		Server: ServerConfig{
			Host: getEnv("LLMPROXY_HOST", "0.0.0.0"),
			Port: getEnv("LLMPROXY_PORT", "9001"),
		},
		Auth: AuthConfig{
			JWTSecret: os.Getenv("JWT_SECRET"),
		},
		OpenAI: OpenAIConfig{
			APIKey:       os.Getenv("OPENAI_API_KEY"),
			BaseURL:      getEnv("OPENAI_BASE_URL", "https://api.openai.com/v1"),
			DefaultModel: getEnv("OPENAI_DEFAULT_MODEL", "gpt-4o"),
		},
		Limits: LimitsConfig{
			MaxTokensPerRequest: getEnvInt("LLMPROXY_MAX_TOKENS", 8192),
			RequestTimeout:      getEnvDuration("LLMPROXY_TIMEOUT", 5*time.Minute),
			StreamingTimeout:    getEnvDuration("LLMPROXY_STREAMING_TIMEOUT", 10*time.Minute),
		},
	}

	// Validate required fields
	if cfg.Auth.JWTSecret == "" {
		return nil, fmt.Errorf("JWT_SECRET environment variable is required")
	}
	if cfg.OpenAI.APIKey == "" {
		return nil, fmt.Errorf("OPENAI_API_KEY environment variable is required")
	}

	return cfg, nil
}

// Address returns the server address in host:port format
func (c *ServerConfig) Address() string {
	return c.Host + ":" + c.Port
}

// IsConfigured returns true if OpenAI is configured
func (c *OpenAIConfig) IsConfigured() bool {
	return c.APIKey != ""
}

// Helper functions

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}
