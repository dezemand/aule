package repository

import (
	"context"

	"github.com/dezemandje/aule/internal/domain"
)

type ProjectRepository interface {
	Create(ctx context.Context, name string, description string) (domain.ProjectID, error)

	FindProjectsForUser(ctx context.Context, userID domain.UserID) ([]domain.Project, []domain.ProjectMember, error)
}
