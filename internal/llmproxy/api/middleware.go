package api

import (
	"strconv"
	"strings"

	"github.com/dezemandje/aule/internal/llmproxy/config"
	"github.com/dezemandje/aule/internal/llmproxy/provider"
	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
)

// LLMConfig holds the parsed LLM configuration from headers
type LLMConfig struct {
	Provider    string
	Model       string
	MaxTokens   int
	Temperature float64
}

// ContextKey constants for Fiber locals
const (
	ContextKeyAgentID   = "agent_id"
	ContextKeyLLMConfig = "llm_config"
)

// JWTMiddleware validates JWT tokens
func JWTMiddleware(cfg *config.AuthConfig) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Get Authorization header
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "missing authorization header",
			})
		}

		// Extract token from "Bearer <token>"
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "invalid authorization header format",
			})
		}

		tokenString := parts[1]

		// Parse and validate token
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			// Validate signing method
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fiber.NewError(fiber.StatusUnauthorized, "invalid signing method")
			}
			return []byte(cfg.JWTSecret), nil
		})

		if err != nil || !token.Valid {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "invalid token",
			})
		}

		// Extract claims
		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "invalid token claims",
			})
		}

		// Verify role is agent
		role, _ := claims["role"].(string)
		if role != "agent" {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "token is not an agent token",
			})
		}

		// Store agent ID in context
		if agentID, ok := claims["id"].(string); ok {
			c.Locals(ContextKeyAgentID, agentID)
		}

		return c.Next()
	}
}

// LLMConfigMiddleware extracts LLM configuration from headers
func LLMConfigMiddleware(registry *provider.Registry, defaultProvider, defaultModel string, defaultMaxTokens int) fiber.Handler {
	return func(c *fiber.Ctx) error {
		cfg := LLMConfig{
			Provider:    c.Get("X-LLM-Provider", defaultProvider),
			Model:       c.Get("X-LLM-Model", defaultModel),
			MaxTokens:   defaultMaxTokens,
			Temperature: 0.0,
		}

		// Parse max tokens if provided
		if maxTokensStr := c.Get("X-LLM-Max-Tokens"); maxTokensStr != "" {
			if maxTokens, err := strconv.Atoi(maxTokensStr); err == nil && maxTokens > 0 {
				cfg.MaxTokens = maxTokens
			}
		}

		// Parse temperature if provided
		if tempStr := c.Get("X-LLM-Temperature"); tempStr != "" {
			if temp, err := strconv.ParseFloat(tempStr, 64); err == nil && temp >= 0 && temp <= 2 {
				cfg.Temperature = temp
			}
		}

		// Validate provider exists
		_, err := registry.Get(cfg.Provider)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": err.Error(),
			})
		}

		c.Locals(ContextKeyLLMConfig, cfg)
		return c.Next()
	}
}

// GetLLMConfig retrieves the LLM config from context
func GetLLMConfig(c *fiber.Ctx) LLMConfig {
	if cfg, ok := c.Locals(ContextKeyLLMConfig).(LLMConfig); ok {
		return cfg
	}
	return LLMConfig{}
}

// GetAgentID retrieves the agent ID from context
func GetAgentID(c *fiber.Ctx) string {
	if id, ok := c.Locals(ContextKeyAgentID).(string); ok {
		return id
	}
	return ""
}
