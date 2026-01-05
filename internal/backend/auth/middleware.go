package auth

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/log"
)

type AuthMiddleware struct {
	service *AuthService
}

func NewAuthMiddleware(authService *AuthService) *AuthMiddleware {
	return &AuthMiddleware{
		service: authService,
	}
}

func (am *AuthMiddleware) require(c *fiber.Ctx, token string, role AuthRole) error {
	auth, err := am.service.VerifyJWT(token)
	if err != nil {
		log.Errorf("failed to verify JWT: %v", err)
		return fiber.ErrUnauthorized
	}

	if auth.Role() != role {
		return fiber.ErrForbidden
	}

	c.Locals("auth", auth)
	return c.Next()
}

func (am *AuthMiddleware) RequireUser(c *fiber.Ctx) error {
	authHeader := c.Get("Authorization")
	if authHeader == "" {
		return fiber.ErrUnauthorized
	}

	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || parts[0] != "Bearer" {
		return fiber.ErrBadRequest
	}

	return am.require(c, parts[1], RoleUser)
}

func (am *AuthMiddleware) RequireUserQuery(c *fiber.Ctx) error {
	token := c.Query("token")
	if token == "" {
		return fiber.ErrUnauthorized
	}

	return am.require(c, token, RoleUser)
}

func (am *AuthMiddleware) RequireAgent(c *fiber.Ctx) error {
	authHeader := c.Get("Authorization")
	if authHeader == "" {
		return fiber.ErrUnauthorized
	}

	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || parts[0] != "Bearer" {
		return fiber.ErrBadRequest
	}

	return am.require(c, parts[1], RoleAgent)
}

func GetUser(c *fiber.Ctx) *UserToken {
	auth := c.Locals("auth").(AuthToken)
	if auth == nil || auth.Role() != RoleUser {
		return nil
	}
	return auth.(*UserToken)
}

func GetAgent(c *fiber.Ctx) *AgentToken {
	auth := c.Locals("auth").(AuthToken)
	if auth == nil || auth.Role() != RoleAgent {
		return nil
	}
	return auth.(*AgentToken)
}
