package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/dezemandje/aule/internal/backend/config"
	"github.com/dezemandje/aule/internal/repository"
	"github.com/gofiber/fiber/v2/log"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/oauth2"
)

var ErrUnknownOAuthProvider = errors.New("unknown oauth provider")

type AuthService struct {
	config                 *config.AuthConfig
	oauth                  map[string]*config.OAuthConfig
	refreshTokenRepository repository.RefreshTokenRepository
}

type AuthProviderDescripton struct {
	ID   string
	Name string
}

type RefreshToken struct {
	Token  string
	Expiry time.Time
}

type userInfoResponse struct {
	Sub   string `json:"sub"`
	Email string `json:"email"`
}

func NewAuthService(config *config.AuthConfig, oauth map[string]*config.OAuthConfig, refreshTokenRepository repository.RefreshTokenRepository) *AuthService {
	return &AuthService{
		config:                 config,
		oauth:                  oauth,
		refreshTokenRepository: refreshTokenRepository,
	}
}

func (s *AuthService) keyFunc(_token *jwt.Token) (any, error) {
	return []byte(s.config.JWTSecret), nil
}

func (s *AuthService) generateOAuthState() string {
	// 128-bit random state for CSRF protection
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		// Extremely unlikely; log and continue (will return zeroed bytes if it happens)
		log.Errorf("failed to generate oauth state: %v", err)
	}
	return base64.RawURLEncoding.EncodeToString(b)
}

func (s *AuthService) generateRefreshToken() string {
	// 256-bit random refresh token
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		// Extremely unlikely; log and continue (will return zeroed bytes if it happens)
		log.Errorf("failed to generate refresh token: %v", err)
	}
	return base64.RawURLEncoding.EncodeToString(b)
}

func (s *AuthService) RefreshAuthToken(refreshToken string) (AuthToken, error) {
	userID, ok := s.refreshTokenRepository.Find(refreshToken)
	if !ok {
		return nil, errors.New("invalid refresh token")
	}

	authToken := newUserToken(userID)
	return authToken, nil
}

func (s *AuthService) SignJWT(authToken AuthToken) (string, error) {
	jwtToken := authToken.Token()
	jwt, err := jwtToken.SignedString([]byte(s.config.JWTSecret))

	if err != nil {
		return "", err
	}
	return jwt, nil
}

func (s *AuthService) VerifyJWT(tokenString string) (AuthToken, error) {
	token, err := jwt.Parse(tokenString, s.keyFunc)
	if err != nil {
		return nil, err
	}

	claims := token.Claims.(jwt.MapClaims)
	exp := time.Unix(int64(claims["exp"].(float64)), 0)

	if exp.Before(time.Now()) {
		return nil, jwt.ErrTokenExpired
	}

	role, ok := ToAuthRole(claims["role"].(string))
	if !ok {
		return nil, jwt.ErrTokenInvalidClaims
	}

	switch role {
	case RoleUser:
		return &UserToken{
			id:      claims["id"].(string),
			expires: exp,
		}, nil
	case RoleAgent:
		return &AgentToken{
			id:      claims["id"].(string),
			expires: exp,
		}, nil
	default:
		return nil, jwt.ErrTokenInvalidClaims
	}
}

func (s *AuthService) GetAuthURL(ctx context.Context, provider string) (string, string, error) {
	oauthConfig, ok := s.oauth[provider]
	if !ok {
		return "", "", ErrUnknownOAuthProvider
	}

	state := s.generateOAuthState()
	verifier := oauth2.GenerateVerifier()
	url := oauthConfig.AuthCodeURL(
		state,
		oauth2.S256ChallengeOption(verifier),
	)

	out := state + "|" + verifier

	return url, out, nil
}

func (s *AuthService) Authenticate(ctx context.Context, provider string, code, state, verifier string) (AuthToken, *RefreshToken, error) {
	oauthConfig, ok := s.oauth[provider]
	if !ok {
		return nil, nil, ErrUnknownOAuthProvider
	}

	token, err := oauthConfig.Exchange(ctx, code, oauth2.VerifierOption(verifier))
	if err != nil {
		log.Infof("OAuth exchange error: %v", err)
		return nil, nil, err
	}

	userInfo, err := s.fetchUserInfo(ctx, provider, token)
	if err != nil {
		log.Infof("Fetch user info error: %v", err)
		return nil, nil, err
	}

	_ = userInfo

	userID := "user-1"

	refreshToken := &RefreshToken{
		Token:  s.generateRefreshToken(),
		Expiry: time.Now().Add(s.config.RefreshExpiration),
	}

	s.refreshTokenRepository.Create(userID, refreshToken.Token)

	authToken := newUserToken(userID)

	return authToken, refreshToken, nil
}

func (s *AuthService) GetProviders(ctx context.Context) []AuthProviderDescripton {
	out := make([]AuthProviderDescripton, 0, len(s.oauth))
	for id := range s.oauth {
		out = append(out, AuthProviderDescripton{
			ID:   id,
			Name: s.oauth[id].Name,
		})
	}
	return out
}

func (s *AuthService) fetchUserInfo(ctx context.Context, provider string, token *oauth2.Token) (*userInfoResponse, error) {
	req, _ := http.NewRequestWithContext(ctx, "GET", s.oauth[provider].UserInfoURL, nil)
	req.Header.Set("Authorization", "Bearer "+token.AccessToken)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var userInfo userInfoResponse
	json.NewDecoder(resp.Body).Decode(&userInfo)
	return &userInfo, nil
}
