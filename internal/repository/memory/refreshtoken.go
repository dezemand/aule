package dbmemory

import (
	"github.com/dezemandje/aule/internal/domain"
	"github.com/google/uuid"
)

type MemoryRefreshTokenRepository struct {
	store map[string]domain.UserID
}

func NewMemoryRefreshTokenRepository() *MemoryRefreshTokenRepository {
	return &MemoryRefreshTokenRepository{
		store: make(map[string]domain.UserID),
	}
}

func (r *MemoryRefreshTokenRepository) Create(userID domain.UserID, token string) error {
	r.store[token] = userID
	return nil
}

func (r *MemoryRefreshTokenRepository) Find(token string) (domain.UserID, bool) {
	userID, ok := r.store[token]
	if !ok {
		return domain.UserID(uuid.Nil), false
	}
	return userID, true
}

func (r *MemoryRefreshTokenRepository) Delete(token string) error {
	delete(r.store, token)
	return nil
}
