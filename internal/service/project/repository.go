package projectsservice

import (
	"context"

	"github.com/dezemandje/aule/internal/domain"
	"github.com/google/uuid"
)

// Repository defines the interface for project persistence.
type Repository interface {
	// Create creates a new project and returns its ID.
	Create(ctx context.Context, project *domain.Project) (domain.ProjectID, error)

	// FindByID returns a project by its ID.
	FindByID(ctx context.Context, id domain.ProjectID) (*domain.Project, error)

	// FindByUserID returns all projects accessible by a user.
	FindByUserID(ctx context.Context, userID domain.UserID) ([]domain.Project, []domain.ProjectMember, error)

	// Update updates a project's fields.
	Update(ctx context.Context, project *domain.Project) (*domain.Project, error)

	// Delete removes a project by its ID.
	Delete(ctx context.Context, id domain.ProjectID) error

	// IsMember checks if a user is a member of a project.
	IsMember(ctx context.Context, projectID domain.ProjectID, userID domain.UserID) (bool, domain.ProjectMemberRole, error)

	// AddMember adds a user as a member of a project.
	AddMember(ctx context.Context, projectID domain.ProjectID, userID domain.UserID, role domain.ProjectMemberRole) error

	// UpdateMember updates a project member's role or permissions.
	UpdateMember(ctx context.Context, projectID domain.ProjectID, userID domain.UserID, role domain.ProjectMemberRole, permissions *domain.ProjectMemberPermissions) error

	// RemoveMember removes a user from a project.
	RemoveMember(ctx context.Context, projectID domain.ProjectID, userID domain.UserID) error

	// ListMembers returns all members of a project.
	ListMembers(ctx context.Context, projectID domain.ProjectID) ([]domain.ProjectMember, error)
}

// RepositoryRepository defines the interface for project repository (git) attachments.
type RepositoryRepository interface {
	// Create adds a repository to a project.
	Create(ctx context.Context, repo *domain.ProjectRepository) error

	// FindByID returns a repository by its ID.
	FindByID(ctx context.Context, id uuid.UUID) (*domain.ProjectRepository, error)

	// FindByProjectID returns all repositories for a project.
	FindByProjectID(ctx context.Context, projectID domain.ProjectID) ([]domain.ProjectRepository, error)

	// Update updates a repository's settings.
	Update(ctx context.Context, repo *domain.ProjectRepository) (*domain.ProjectRepository, error)

	// Delete removes a repository from a project.
	Delete(ctx context.Context, id uuid.UUID) error
}

// WikiSpaceRepository defines the interface for project wiki space attachments.
type WikiSpaceRepository interface {
	// Create adds a wiki space to a project.
	Create(ctx context.Context, space *domain.ProjectWikiSpace) error

	// FindByID returns a wiki space by its ID.
	FindByID(ctx context.Context, id uuid.UUID) (*domain.ProjectWikiSpace, error)

	// FindByProjectID returns all wiki spaces for a project.
	FindByProjectID(ctx context.Context, projectID domain.ProjectID) ([]domain.ProjectWikiSpace, error)

	// Update updates a wiki space's settings.
	Update(ctx context.Context, space *domain.ProjectWikiSpace) (*domain.ProjectWikiSpace, error)

	// Delete removes a wiki space from a project.
	Delete(ctx context.Context, id uuid.UUID) error
}
