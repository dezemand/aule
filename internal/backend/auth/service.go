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
	"github.com/dezemandje/aule/internal/domain"
	"github.com/dezemandje/aule/internal/repository"
	userservice "github.com/dezemandje/aule/internal/service/user"
	"github.com/gofiber/fiber/v2/log"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/oauth2"
)

var ErrUnknownOAuthProvider = errors.New("unknown oauth provider")

type AuthService struct {
	config                 *config.AuthConfig
	oauth                  *map[string]config.OAuthProviderConfig
	refreshTokenRepository RefreshTokenRepository
	userRepository         userservice.Repository
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
	Sub               string   `json:"sub"`
	Email             string   `json:"email"`
	EmailVerified     bool     `json:"email_verified"`
	Name              string   `json:"name"`
	GivenName         string   `json:"given_name"`
	PreferredUsername string   `json:"preferred_username"`
	Nickname          string   `json:"nickname"`
	Groups            []string `json:"groups"`
}

func NewAuthService(config *config.AuthConfig, oauth *map[string]config.OAuthProviderConfig, refreshTokenRepository RefreshTokenRepository, userRepository userservice.Repository) *AuthService {
	return &AuthService{
		config:                 config,
		oauth:                  oauth,
		refreshTokenRepository: refreshTokenRepository,
		userRepository:         userRepository,
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
	userID, err := s.refreshTokenRepository.Find(refreshToken)
	if err != nil {
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
		id := domain.UserID(uuid.MustParse(claims["id"].(string)))
		return &UserToken{
			id:      id,
			expires: exp,
		}, nil
	case RoleAgent:
		id := domain.AgentInstanceID(uuid.MustParse(claims["id"].(string)))
		return &AgentToken{
			id:      id,
			expires: exp,
		}, nil
	default:
		return nil, jwt.ErrTokenInvalidClaims
	}
}

func (s *AuthService) GetAuthURL(ctx context.Context, provider string) (string, string, error) {
	oauthConfig, ok := (*s.oauth)[provider]
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
	oauthConfig, ok := (*s.oauth)[provider]
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

	user, err := s.userRepository.FindByIdentity(provider, userInfo.Sub)
	if err != nil && errors.Is(err, repository.ErrNotFound) {
		user, err = s.createUser(ctx, provider, userInfo)
	}
	if err != nil {
		log.Infof("User retrieval/creation error: %v", err)
		return nil, nil, err
	}

	refreshToken := &RefreshToken{
		Token:  s.generateRefreshToken(),
		Expiry: time.Now().Add(s.config.RefreshExpiration),
	}

	err = s.refreshTokenRepository.Create(user.ID, refreshToken.Token, refreshToken.Expiry)
	authToken := newUserToken(user.ID)

	return authToken, refreshToken, nil
}

func (s *AuthService) GetProviders(ctx context.Context) []AuthProviderDescripton {
	out := make([]AuthProviderDescripton, 0, len(*s.oauth))
	for id := range *s.oauth {
		out = append(out, AuthProviderDescripton{
			ID:   id,
			Name: (*s.oauth)[id].Name,
		})
	}
	return out
}

func (s *AuthService) fetchUserInfo(ctx context.Context, provider string, token *oauth2.Token) (*userInfoResponse, error) {
	req, _ := http.NewRequestWithContext(ctx, "GET", (*s.oauth)[provider].UserInfoURL, nil)
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

func (s *AuthService) createUser(ctx context.Context, provider string, userInfo *userInfoResponse) (*domain.User, error) {
	user := &domain.User{
		Email: userInfo.Email,
		Name:  userInfo.Name,
	}

	id, err := s.userRepository.Create(user)
	if err != nil {
		return nil, err
	}

	err = s.userRepository.AddIdentity(*id, provider, userInfo.Sub)
	if err != nil {
		return nil, err
	}

	return user, nil
}

func (s *AuthService) RevokeRefreshToken(refreshToken string) error {
	err := s.refreshTokenRepository.Delete(refreshToken)
	return err
}
