package auth

import (
	"time"

	"github.com/dezemandje/aule/internal/domain"
)

type RefreshTokenRepository interface {
	// Creates a new refresh token for the given user ID with an expiration time.
	Create(userID domain.UserID, token string, expires time.Time) error

	// Finds the user ID associated with the given refresh token.
	Find(token string) (domain.UserID, error)

	// Deletes the given refresh token.
	Delete(token string) error
}
