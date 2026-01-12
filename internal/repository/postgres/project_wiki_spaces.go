package postgres

import (
	"context"
	"database/sql"
	"errors"

	"github.com/dezemandje/aule/internal/database"
	"github.com/dezemandje/aule/internal/domain"
	projectsservice "github.com/dezemandje/aule/internal/service/project"
	"github.com/google/uuid"
	"github.com/lib/pq"
)

var ErrWikiSpaceNotFound = errors.New("wiki space not found")

type postgresWikiSpaceRepository struct {
	db *database.DB
}

func NewProjectWikiSpaceRepository(db *database.DB) projectsservice.WikiSpaceRepository {
	return &postgresWikiSpaceRepository{db: db}
}

func (r *postgresWikiSpaceRepository) Create(ctx context.Context, space *domain.ProjectWikiSpace) error {
	query := `
		INSERT INTO aule.project_wiki_spaces (project_id, space_id, access_mode, page_prefixes)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at, updated_at
	`

	accessMode := space.AccessMode
	if accessMode == "" {
		accessMode = "read"
	}

	err := r.db.QueryRowContext(ctx, query,
		uuid.UUID(space.ProjectID),
		space.SpaceID,
		accessMode,
		pq.Array(space.PagePrefixes),
	).Scan(&space.ID, &space.CreatedAt, &space.UpdatedAt)

	return err
}

func (r *postgresWikiSpaceRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.ProjectWikiSpace, error) {
	query := `
		SELECT id, project_id, space_id, access_mode, page_prefixes, created_at, updated_at
		FROM aule.project_wiki_spaces
		WHERE id = $1
	`

	var space domain.ProjectWikiSpace
	var projectID uuid.UUID
	var pagePrefixes []string

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&space.ID,
		&projectID,
		&space.SpaceID,
		&space.AccessMode,
		pq.Array(&pagePrefixes),
		&space.CreatedAt,
		&space.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	space.ProjectID = domain.ProjectID(projectID)
	space.PagePrefixes = pagePrefixes

	return &space, nil
}

func (r *postgresWikiSpaceRepository) FindByProjectID(ctx context.Context, projectID domain.ProjectID) ([]domain.ProjectWikiSpace, error) {
	query := `
		SELECT id, project_id, space_id, access_mode, page_prefixes, created_at, updated_at
		FROM aule.project_wiki_spaces
		WHERE project_id = $1
		ORDER BY created_at ASC
	`

	rows, err := r.db.QueryContext(ctx, query, uuid.UUID(projectID))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var spaces []domain.ProjectWikiSpace
	for rows.Next() {
		var space domain.ProjectWikiSpace
		var pID uuid.UUID
		var pagePrefixes []string

		err := rows.Scan(
			&space.ID,
			&pID,
			&space.SpaceID,
			&space.AccessMode,
			pq.Array(&pagePrefixes),
			&space.CreatedAt,
			&space.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		space.ProjectID = domain.ProjectID(pID)
		space.PagePrefixes = pagePrefixes
		spaces = append(spaces, space)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return spaces, nil
}

func (r *postgresWikiSpaceRepository) Update(ctx context.Context, space *domain.ProjectWikiSpace) (*domain.ProjectWikiSpace, error) {
	query := `
		UPDATE aule.project_wiki_spaces
		SET
			access_mode = $2,
			page_prefixes = $3,
			updated_at = NOW()
		WHERE id = $1
		RETURNING id, project_id, space_id, access_mode, page_prefixes, created_at, updated_at
	`

	var updated domain.ProjectWikiSpace
	var projectID uuid.UUID
	var pagePrefixes []string

	err := r.db.QueryRowContext(ctx, query,
		space.ID,
		space.AccessMode,
		pq.Array(space.PagePrefixes),
	).Scan(
		&updated.ID,
		&projectID,
		&updated.SpaceID,
		&updated.AccessMode,
		pq.Array(&pagePrefixes),
		&updated.CreatedAt,
		&updated.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrWikiSpaceNotFound
	}
	if err != nil {
		return nil, err
	}

	updated.ProjectID = domain.ProjectID(projectID)
	updated.PagePrefixes = pagePrefixes

	return &updated, nil
}

func (r *postgresWikiSpaceRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM aule.project_wiki_spaces WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return ErrWikiSpaceNotFound
	}

	return nil
}
