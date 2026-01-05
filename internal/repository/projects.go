package repository

import "context"

type ProjectRepository interface {
	Create(ctx context.Context, name string, description string) (string, error)
}
