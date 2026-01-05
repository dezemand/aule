package config

import "golang.org/x/oauth2"

type OAuthConfig struct {
	oauth2.Config

	ID          string
	Name        string
	UserInfoURL string
}

func NewOAuthConfigFromEnv(root string, id string) *OAuthConfig {
	return &OAuthConfig{
		Config: oauth2.Config{
			ClientID:     getEnv(root+"_CLIENT_ID", ""),
			ClientSecret: getEnv(root+"_CLIENT_SECRET", ""),
			Endpoint: oauth2.Endpoint{
				AuthURL:  getEnv(root+"_AUTH_URL", ""),
				TokenURL: getEnv(root+"_TOKEN_URL", ""),
			},
			RedirectURL: getEnv(root+"_REDIRECT_URL", ""),
			Scopes:      getEnvStringSlice(root+"_SCOPES", []string{"email", "profile"}),
		},
		ID:          id,
		Name:        getEnv(root+"_NAME", ""),
		UserInfoURL: getEnv(root+"_USERINFO_URL", ""),
	}
}
