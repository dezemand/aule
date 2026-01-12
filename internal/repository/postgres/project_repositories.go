package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"

	"github.com/dezemandje/aule/internal/database"
	"github.com/dezemandje/aule/internal/domain"
	projectsservice "github.com/dezemandje/aule/internal/service/project"
	"github.com/google/uuid"
)

var ErrRepositoryNotFound = errors.New("repository not found")

type postgresRepositoryRepository struct {
	db *database.DB
}

func NewProjectRepositoryRepository(db *database.DB) projectsservice.RepositoryRepository {
	return &postgresRepositoryRepository{db: db}
}

func (r *postgresRepositoryRepository) Create(ctx context.Context, repo *domain.ProjectRepository) error {
	query := `
		INSERT INTO aule.project_repositories (project_id, url, purpose, default_branch, allowed_paths, branch_naming_convention)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at, updated_at
	`

	allowedPathsJSON, err := marshalJSONB(repo.AllowedPaths)
	if err != nil {
		return err
	}

	defaultBranch := repo.DefaultBranch
	if defaultBranch == "" {
		defaultBranch = "main"
	}

	err = r.db.QueryRowContext(ctx, query,
		uuid.UUID(repo.ProjectID),
		repo.URL,
		repo.Purpose,
		defaultBranch,
		allowedPathsJSON,
		nullString(repo.BranchNamingConvention),
	).Scan(&repo.ID, &repo.CreatedAt, &repo.UpdatedAt)

	return err
}

func (r *postgresRepositoryRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.ProjectRepository, error) {
	query := `
		SELECT id, project_id, url, purpose, default_branch, allowed_paths, branch_naming_convention, created_at, updated_at
		FROM aule.project_repositories
		WHERE id = $1
	`

	var repo domain.ProjectRepository
	var projectID uuid.UUID
	var allowedPathsJSON []byte
	var branchNaming sql.NullString

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&repo.ID,
		&projectID,
		&repo.URL,
		&repo.Purpose,
		&repo.DefaultBranch,
		&allowedPathsJSON,
		&branchNaming,
		&repo.CreatedAt,
		&repo.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	repo.ProjectID = domain.ProjectID(projectID)
	if branchNaming.Valid {
		repo.BranchNamingConvention = branchNaming.String
	}
	if err := unmarshalJSONB(allowedPathsJSON, &repo.AllowedPaths); err != nil {
		return nil, err
	}

	return &repo, nil
}

func (r *postgresRepositoryRepository) FindByProjectID(ctx context.Context, projectID domain.ProjectID) ([]domain.ProjectRepository, error) {
	query := `
		SELECT id, project_id, url, purpose, default_branch, allowed_paths, branch_naming_convention, created_at, updated_at
		FROM aule.project_repositories
		WHERE project_id = $1
		ORDER BY created_at ASC
	`

	rows, err := r.db.QueryContext(ctx, query, uuid.UUID(projectID))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var repos []domain.ProjectRepository
	for rows.Next() {
		var repo domain.ProjectRepository
		var pID uuid.UUID
		var allowedPathsJSON []byte
		var branchNaming sql.NullString

		err := rows.Scan(
			&repo.ID,
			&pID,
			&repo.URL,
			&repo.Purpose,
			&repo.DefaultBranch,
			&allowedPathsJSON,
			&branchNaming,
			&repo.CreatedAt,
			&repo.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		repo.ProjectID = domain.ProjectID(pID)
		if branchNaming.Valid {
			repo.BranchNamingConvention = branchNaming.String
		}
		if err := unmarshalJSONB(allowedPathsJSON, &repo.AllowedPaths); err != nil {
			return nil, err
		}

		repos = append(repos, repo)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return repos, nil
}

func (r *postgresRepositoryRepository) Update(ctx context.Context, repo *domain.ProjectRepository) (*domain.ProjectRepository, error) {
	query := `
		UPDATE aule.project_repositories
		SET
			purpose = $2,
			default_branch = $3,
			allowed_paths = $4,
			branch_naming_convention = $5,
			updated_at = NOW()
		WHERE id = $1
		RETURNING id, project_id, url, purpose, default_branch, allowed_paths, branch_naming_convention, created_at, updated_at
	`

	allowedPathsJSON, err := marshalJSONB(repo.AllowedPaths)
	if err != nil {
		return nil, err
	}

	var updated domain.ProjectRepository
	var projectID uuid.UUID
	var allowedPathsOut []byte
	var branchNaming sql.NullString

	err = r.db.QueryRowContext(ctx, query,
		repo.ID,
		repo.Purpose,
		repo.DefaultBranch,
		allowedPathsJSON,
		nullString(repo.BranchNamingConvention),
	).Scan(
		&updated.ID,
		&projectID,
		&updated.URL,
		&updated.Purpose,
		&updated.DefaultBranch,
		&allowedPathsOut,
		&branchNaming,
		&updated.CreatedAt,
		&updated.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrRepositoryNotFound
	}
	if err != nil {
		return nil, err
	}

	updated.ProjectID = domain.ProjectID(projectID)
	if branchNaming.Valid {
		updated.BranchNamingConvention = branchNaming.String
	}
	if err := unmarshalJSONB(allowedPathsOut, &updated.AllowedPaths); err != nil {
		return nil, err
	}

	return &updated, nil
}

func (r *postgresRepositoryRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM aule.project_repositories WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return ErrRepositoryNotFound
	}

	return nil
}

// Helper to unmarshal JSONB into AllowedPaths specifically
func unmarshalAllowedPaths(data []byte) (*domain.AllowedPaths, error) {
	if data == nil || len(data) == 0 {
		return nil, nil
	}
	var paths domain.AllowedPaths
	if err := json.Unmarshal(data, &paths); err != nil {
		return nil, err
	}
	return &paths, nil
}
