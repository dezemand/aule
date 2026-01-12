package postgres

import (
	"context"
	"database/sql"
	"errors"

	"github.com/dezemandje/aule/internal/database"
	"github.com/dezemandje/aule/internal/domain"
	"github.com/dezemandje/aule/internal/repository"
	userservice "github.com/dezemandje/aule/internal/service/user"
	"github.com/google/uuid"
)

type postgresUserRepository struct {
	db *database.DB
}

// Hello implements UserRepository.
func (r *postgresUserRepository) Hello() string {
	panic("unimplemented")
}

func NewUserRepository(db *database.DB) userservice.Repository {
	return &postgresUserRepository{db: db}
}

func (r *postgresUserRepository) Create(user *domain.User) (*domain.UserID, error) {
	query := `
		INSERT INTO aule.users (email, name)
		VALUES ($1, $2)
		RETURNING id
	`

	var idStr string
	err := r.db.QueryRowContext(context.Background(), query, user.Email, user.Name).Scan(&idStr)
	if err != nil {
		return nil, err
	}

	id, err := uuid.Parse(idStr)
	if err != nil {
		return nil, err
	}

	user.ID = domain.UserID(id)
	return &user.ID, nil
}

func (r *postgresUserRepository) Update(user *domain.User) error {
	query := `
		UPDATE aule.users
		SET email = $2, name = $3
		WHERE id = $1
	`

	result, err := r.db.Exec(query, uuid.UUID(user.ID), user.Email, user.Name)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return repository.ErrNotFound
	}

	return nil
}

func (r *postgresUserRepository) FindByIdentity(provider string, sub string) (*domain.User, error) {
	query := `
		SELECT u.id, u.email, u.name
		FROM aule.users u
		INNER JOIN aule.user_identities ui ON u.id = ui.user_id
		WHERE ui.provider = $1 AND ui.sub = $2
	`

	var id uuid.UUID
	var email, name sql.NullString

	err := r.db.QueryRowContext(context.Background(), query, provider, sub).Scan(&id, &email, &name)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, repository.ErrNotFound
		}
		return nil, err
	}

	return &domain.User{
		ID:    domain.UserID(id),
		Email: email.String,
		Name:  name.String,
	}, nil
}

func (r *postgresUserRepository) AddIdentity(userID domain.UserID, provider string, sub string) error {
	query := `
		INSERT INTO aule.user_identities (user_id, provider, sub)
		VALUES ($1, $2, $3)
	`

	_, err := r.db.Exec(query, uuid.UUID(userID), provider, sub)
	return err
}
