package userservice

import "github.com/dezemandje/aule/internal/domain"

type Repository interface {
	Create(user *domain.User) (*domain.UserID, error)
	Update(user *domain.User) error

	FindByIdentity(provider string, sub string) (*domain.User, error)
	AddIdentity(userID domain.UserID, provider string, sub string) error
}
