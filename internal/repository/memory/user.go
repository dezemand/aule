package dbmemory

import (
	"github.com/dezemandje/aule/internal/domain"
	"github.com/dezemandje/aule/internal/repository"
	"github.com/gofiber/fiber/v2/log"
	"github.com/google/uuid"
)

type loginKey struct {
	provider string
	sub      string
}

type MemoryUserRepository struct {
	users  map[domain.UserID]domain.User
	logins map[loginKey]domain.UserLogin
}

func NewMemoryUserRepository() *MemoryUserRepository {
	return &MemoryUserRepository{
		users:  make(map[domain.UserID]domain.User),
		logins: make(map[loginKey]domain.UserLogin),
	}
}

// Create(user *domain.User) (*domain.UserID, error)
// Update(user *domain.User) error

// FindBySub(provider string, sub string) (*domain.User, error)
// AddSub(userID domain.UserID, provider string, sub string) error

func (r *MemoryUserRepository) Create(user *domain.User) (*domain.UserID, error) {
	log.Infof("Create user %v", user)

	userID := domain.UserID(uuid.New())
	user.ID = userID
	r.users[userID] = *user
	return &userID, nil
}

func (r *MemoryUserRepository) Update(user *domain.User) error {
	log.Infof("Update user %v", user)

	r.users[user.ID] = *user
	return nil
}

func (r *MemoryUserRepository) FindByIdentity(provider string, sub string) (*domain.User, error) {
	log.Infof("FindBySub %v %v", provider, sub)

	key := loginKey{provider: provider, sub: sub}
	login, ok := r.logins[key]
	if !ok {
		return nil, repository.ErrNotFound
	}
	user, ok := r.users[login.UserID]
	if !ok {
		return nil, repository.ErrNotFound
	}
	return &user, nil
}

func (r *MemoryUserRepository) AddIdentity(userID domain.UserID, provider string, sub string) error {
	log.Infof("AddSub %v %v %v", userID, provider, sub)

	key := loginKey{provider: provider, sub: sub}
	r.logins[key] = domain.UserLogin{
		UserID:   userID,
		Provider: provider,
		Sub:      sub,
	}
	return nil
}
