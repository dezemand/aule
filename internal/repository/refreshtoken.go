package repository

import "github.com/dezemandje/aule/internal/domain"

type RefreshTokenRepository interface {
	Create(userID domain.UserID, token string) error
	Find(token string) (domain.UserID, bool)
	Delete(token string) error
}
