package repository

import "github.com/dezemandje/aule/internal/domain"

type UserRepository interface {
	Create(user *domain.User) (*domain.UserID, error)
	Update(user *domain.User) error

	FindBySub(provider string, sub string) (*domain.User, error)
	AddSub(userID domain.UserID, provider string, sub string) error
}
