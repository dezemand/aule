package userrest

import (
	"errors"
	"strings"
	"time"

	"github.com/dezemandje/aule/internal/backend/auth"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/log"
)

const RefreshTokenCookie = "t"
const OAuthStateCookie = "ostate"

type AuthHandler struct {
	service *auth.AuthService
}

func NewAuthHandler(authService *auth.AuthService) *AuthHandler {
	return &AuthHandler{
		service: authService,
	}
}

type AuthResponse struct {
	Token   string `json:"token"`
	Expires string `json:"expires,omitempty"`
}

// GetJWT issues a new JWT based on a valid refresh token cookie.
//
// Requires:
//   - Refresh token cookie
//
// Returns:
//   - 200: AuthResponse
//   - 401: Unauthorized (missing or invalid refresh token)
//   - 500: Internal Server Error
func (h *AuthHandler) GetJWT(c *fiber.Ctx) error {
	refreshToken := c.Cookies(RefreshTokenCookie)

	if refreshToken == "" {
		return fiber.ErrUnauthorized
	}

	auth, err := h.service.RefreshAuthToken(refreshToken)
	if err != nil {
		log.Errorf("failed to refresh auth token: %v", err)
		return fiber.ErrUnauthorized
	}

	token, err := h.service.SignJWT(auth)
	if err != nil {
		log.Errorf("failed to sign jwt: %v", err)
		return fiber.ErrInternalServerError
	}

	return c.JSON(&AuthResponse{Token: token})
}

// StartOAuthResponse contains the OAuth start response. Rather than redirecting, we let the
// frontend handle it.
type StartOAuthResponse struct {
	AuthURL string `json:"authUrl"`
}

// StartOAuth initiates the OAuth flow.
//
// Requires:
//   - provider: path param
//
// Returns:
//   - 200: StartOAuthResponse
//   - 400: Bad Request (unknown provider)
//   - 500: Internal Server Error
func (h *AuthHandler) StartOAuth(c *fiber.Ctx) error {
	url, cookie, err := h.service.GetAuthURL(c.UserContext(), c.Params("provider"))
	if err != nil {
		if errors.Is(err, auth.ErrUnknownOAuthProvider) {
			return fiber.ErrBadRequest
		}

		log.Errorf("failed to get auth code url: %v", err)
		return fiber.ErrInternalServerError
	}

	c.Cookie(&fiber.Cookie{
		Name:     OAuthStateCookie,
		Value:    cookie,
		HTTPOnly: true,
		Expires:  c.Context().Time().Add(time.Minute * 5),
	})

	return c.JSON(&StartOAuthResponse{
		AuthURL: url,
	})
}

// CallbackOAuth handles the OAuth callback
//
// Requires:
//   - provider: path param
//   - Query params: state, code
//
// Returns:
//   - 200: AuthResponse
//   - 400: Bad Request (missing params)
//   - 401: Unauthorized (invalid state)
//   - 500: Internal Server Error
func (h *AuthHandler) CallbackOAuth(c *fiber.Ctx) error {
	state := c.Query("state")
	if state == "" {
		return fiber.ErrBadRequest
	}
	stateCookieVal := c.Cookies(OAuthStateCookie)
	if stateCookieVal == "" {
		return fiber.ErrUnauthorized
	}

	parts := strings.Split(stateCookieVal, "|")
	stateCookie := parts[0]
	verifier := parts[1]

	if state != stateCookie {
		return fiber.ErrUnauthorized
	}

	code := c.Query("code")
	if code == "" {
		return fiber.ErrBadRequest
	}

	authToken, refreshToken, err := h.service.Authenticate(
		c.UserContext(),
		c.Params("provider"),
		code,
		state,
		verifier,
	)
	if err != nil {
		log.Errorf("failed to authenticate: %v", err)

		return fiber.ErrUnauthorized
	}

	c.Cookie(&fiber.Cookie{
		Name:     RefreshTokenCookie,
		Value:    refreshToken.Token,
		HTTPOnly: true,
		Expires:  refreshToken.Expiry,
	})

	token, err := h.service.SignJWT(authToken)
	if err != nil {
		log.Errorf("failed to sign jwt: %v", err)
		return fiber.ErrInternalServerError
	}

	return c.JSON(&AuthResponse{
		Token:   token,
		Expires: refreshToken.Expiry.Format(time.RFC3339),
	})
}

type AuthProvidersResponse struct {
	Providers []AuthProvider `json:"providers"`
}

type AuthProvider struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func (h *AuthHandler) GetProviders(c *fiber.Ctx) error {
	providers := h.service.GetProviders(c.UserContext())

	providersList := make([]AuthProvider, 0, len(providers))
	for _, p := range providers {
		providersList = append(providersList, AuthProvider{
			ID:   p.ID,
			Name: p.Name,
		})
	}

	return c.JSON(&AuthProvidersResponse{
		Providers: providersList,
	})
}

func (h *AuthHandler) Logout(c *fiber.Ctx) error {
	refreshTokenCookie := c.Cookies(RefreshTokenCookie)
	if refreshTokenCookie != "" {
		_ = h.service.RevokeRefreshToken(refreshTokenCookie)
	}

	c.Cookie(&fiber.Cookie{
		Name:     RefreshTokenCookie,
		Value:    "",
		HTTPOnly: true,
		Expires:  time.Unix(0, 0),
	})

	return c.SendStatus(fiber.StatusNoContent)
}
