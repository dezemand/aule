package postgres

import (
	"context"

	"github.com/dezemandje/aule/internal/database"
	"github.com/dezemandje/aule/internal/domain"
	projectsservice "github.com/dezemandje/aule/internal/service/project"
	"github.com/google/uuid"
)

type postgresProjectRepository struct {
	db *database.DB
}

func NewProjectRepository(db *database.DB) projectsservice.Repository {
	return &postgresProjectRepository{db: db}
}

func (r *postgresProjectRepository) Create(ctx context.Context, name string, description string) (domain.ProjectID, error) {
	query := `
		INSERT INTO aule.projects (key, name, description)
		VALUES ($1, $2, $3)
		RETURNING id
	`

	// Generate a unique key from UUID (first 8 chars)
	key := uuid.New().String()[:8]

	var id uuid.UUID
	err := r.db.QueryRowContext(ctx, query, key, name, description).Scan(&id)
	if err != nil {
		return domain.ProjectID(uuid.Nil), err
	}

	return domain.ProjectID(id), nil
}

func (r *postgresProjectRepository) FindProjectsForUser(ctx context.Context, userID domain.UserID) ([]domain.Project, []domain.ProjectMember, error) {
	query := `
		SELECT
			p.id, p.key, p.name, p.description, p.goal, p.created_at, p.updated_at,
			pm.id, pm.project_id, pm.user_id, pm.role
		FROM aule.projects p
		INNER JOIN aule.project_members pm ON p.id = pm.project_id
		WHERE pm.user_id = $1
		ORDER BY p.created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, uuid.UUID(userID))
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	var projects []domain.Project
	var members []domain.ProjectMember

	for rows.Next() {
		var p domain.Project
		var m domain.ProjectMember
		var projectID, memberID, memberProjectID, memberUserID uuid.UUID
		var description, goal *string

		err := rows.Scan(
			&projectID, &p.Key, &p.Name, &description, &goal, &p.CreatedAt, &p.UpdatedAt,
			&memberID, &memberProjectID, &memberUserID, &m.Role,
		)
		if err != nil {
			return nil, nil, err
		}

		p.ID = domain.ProjectID(projectID)
		if description != nil {
			p.Description = *description
		}
		if goal != nil {
			p.Goal = *goal
		}

		m.ID = memberID
		m.ProjectID = domain.ProjectID(memberProjectID)
		m.UserID = domain.UserID(memberUserID)

		projects = append(projects, p)
		members = append(members, m)
	}

	if err := rows.Err(); err != nil {
		return nil, nil, err
	}

	return projects, members, nil
}
