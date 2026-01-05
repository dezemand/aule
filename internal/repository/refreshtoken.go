package repository

type RefreshTokenRepository interface {
	Create(userID string, token string) error
	Find(token string) (string, bool)
	Delete(token string) error
}
