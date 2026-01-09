package postgres

import (
	"database/sql"
	"errors"
	"time"

	"github.com/dezemandje/aule/internal/backend/auth"
	"github.com/dezemandje/aule/internal/database"
	"github.com/dezemandje/aule/internal/domain"
	"github.com/dezemandje/aule/internal/repository"
	"github.com/gofiber/fiber/v2/log"
	"github.com/google/uuid"
)

type postgresRefreshTokenRepository struct {
	db *database.DB
}

func NewRefreshTokenRepository(db *database.DB) auth.RefreshTokenRepository {
	return &postgresRefreshTokenRepository{db: db}
}

func (r *postgresRefreshTokenRepository) Create(userID domain.UserID, token string, expires time.Time) error {
	log.Infof("Creating refresh token for user %s with expiration %s", userID.String(), expires)

	query := `
		INSERT INTO aule.refresh_tokens (token, user_id, expires_at)
		VALUES ($1, $2, $3)
	`

	_, err := r.db.Exec(query, token, uuid.UUID(userID), expires)
	return err
}

func (r *postgresRefreshTokenRepository) Find(token string) (domain.UserID, error) {
	query := `
		SELECT user_id
		FROM aule.refresh_tokens
		WHERE token = $1 AND expires_at > NOW()
	`

	var userID uuid.UUID
	err := r.db.QueryRow(query, token).Scan(&userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.UserID(uuid.Nil), repository.ErrNotFound
		}
		return domain.UserID(uuid.Nil), err
	}

	return domain.UserID(userID), nil
}

func (r *postgresRefreshTokenRepository) Delete(token string) error {
	query := `
		DELETE FROM aule.refresh_tokens
		WHERE token = $1
	`

	_, err := r.db.Exec(query, token)
	return err
}
