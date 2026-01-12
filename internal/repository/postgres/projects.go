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
	"github.com/lib/pq"
)

type postgresProjectRepository struct {
	db *database.DB
}

func NewProjectRepository(db *database.DB) projectsservice.Repository {
	return &postgresProjectRepository{db: db}
}

func (r *postgresProjectRepository) Create(ctx context.Context, project *domain.Project) (domain.ProjectID, error) {
	query := `
		INSERT INTO aule.projects (key, name, description, status, purpose, scope, governance, task_config, agent_config)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id
	`

	// Generate a unique key from UUID (first 8 chars) if not provided
	key := project.Key
	if key == "" {
		key = uuid.New().String()[:8]
	}

	// Set default status if not provided
	status := project.Status
	if status == "" {
		status = domain.ProjectStatusActive
	}

	// Marshal JSONB fields
	purposeJSON, err := marshalJSONB(project.Purpose)
	if err != nil {
		return domain.ProjectID(uuid.Nil), err
	}
	scopeJSON, err := marshalJSONB(project.Scope)
	if err != nil {
		return domain.ProjectID(uuid.Nil), err
	}
	governanceJSON, err := marshalJSONB(project.Governance)
	if err != nil {
		return domain.ProjectID(uuid.Nil), err
	}
	taskConfigJSON, err := marshalJSONB(project.TaskConfig)
	if err != nil {
		return domain.ProjectID(uuid.Nil), err
	}
	agentConfigJSON, err := marshalJSONB(project.AgentConfig)
	if err != nil {
		return domain.ProjectID(uuid.Nil), err
	}

	var id uuid.UUID
	err = r.db.QueryRowContext(ctx, query,
		key,
		project.Name,
		nullString(project.Description),
		status,
		purposeJSON,
		scopeJSON,
		governanceJSON,
		taskConfigJSON,
		agentConfigJSON,
	).Scan(&id)
	if err != nil {
		return domain.ProjectID(uuid.Nil), err
	}

	return domain.ProjectID(id), nil
}

func (r *postgresProjectRepository) FindByID(ctx context.Context, id domain.ProjectID) (*domain.Project, error) {
	query := `
		SELECT id, key, name, description, status, purpose, scope, governance, task_config, agent_config, created_at, updated_at
		FROM aule.projects
		WHERE id = $1
	`

	var p domain.Project
	var projectID uuid.UUID
	var description sql.NullString
	var purposeJSON, scopeJSON, governanceJSON, taskConfigJSON, agentConfigJSON []byte

	err := r.db.QueryRowContext(ctx, query, uuid.UUID(id)).Scan(
		&projectID,
		&p.Key,
		&p.Name,
		&description,
		&p.Status,
		&purposeJSON,
		&scopeJSON,
		&governanceJSON,
		&taskConfigJSON,
		&agentConfigJSON,
		&p.CreatedAt,
		&p.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	p.ID = domain.ProjectID(projectID)
	if description.Valid {
		p.Description = description.String
	}

	// Unmarshal JSONB fields
	if err := unmarshalJSONB(purposeJSON, &p.Purpose); err != nil {
		return nil, err
	}
	if err := unmarshalJSONB(scopeJSON, &p.Scope); err != nil {
		return nil, err
	}
	if err := unmarshalJSONB(governanceJSON, &p.Governance); err != nil {
		return nil, err
	}
	if err := unmarshalJSONB(taskConfigJSON, &p.TaskConfig); err != nil {
		return nil, err
	}
	if err := unmarshalJSONB(agentConfigJSON, &p.AgentConfig); err != nil {
		return nil, err
	}

	return &p, nil
}

func (r *postgresProjectRepository) FindByUserID(ctx context.Context, userID domain.UserID) ([]domain.Project, []domain.ProjectMember, error) {
	query := `
		SELECT
			p.id, p.key, p.name, p.description, p.status, p.purpose, p.scope, p.governance, p.task_config, p.agent_config, p.created_at, p.updated_at,
			pm.id, pm.project_id, pm.user_id, pm.role, pm.permissions, pm.created_at
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
		var description sql.NullString
		var purposeJSON, scopeJSON, governanceJSON, taskConfigJSON, agentConfigJSON, permissionsJSON []byte

		err := rows.Scan(
			&projectID, &p.Key, &p.Name, &description, &p.Status, &purposeJSON, &scopeJSON, &governanceJSON, &taskConfigJSON, &agentConfigJSON, &p.CreatedAt, &p.UpdatedAt,
			&memberID, &memberProjectID, &memberUserID, &m.Role, &permissionsJSON, &m.CreatedAt,
		)
		if err != nil {
			return nil, nil, err
		}

		p.ID = domain.ProjectID(projectID)
		if description.Valid {
			p.Description = description.String
		}

		// Unmarshal JSONB fields for project
		if err := unmarshalJSONB(purposeJSON, &p.Purpose); err != nil {
			return nil, nil, err
		}
		if err := unmarshalJSONB(scopeJSON, &p.Scope); err != nil {
			return nil, nil, err
		}
		if err := unmarshalJSONB(governanceJSON, &p.Governance); err != nil {
			return nil, nil, err
		}
		if err := unmarshalJSONB(taskConfigJSON, &p.TaskConfig); err != nil {
			return nil, nil, err
		}
		if err := unmarshalJSONB(agentConfigJSON, &p.AgentConfig); err != nil {
			return nil, nil, err
		}

		m.ID = memberID
		m.ProjectID = domain.ProjectID(memberProjectID)
		m.UserID = domain.UserID(memberUserID)
		if err := unmarshalJSONB(permissionsJSON, &m.Permissions); err != nil {
			return nil, nil, err
		}

		projects = append(projects, p)
		members = append(members, m)
	}

	if err := rows.Err(); err != nil {
		return nil, nil, err
	}

	return projects, members, nil
}

func (r *postgresProjectRepository) Update(ctx context.Context, project *domain.Project) (*domain.Project, error) {
	query := `
		UPDATE aule.projects
		SET
			name = $2,
			description = $3,
			status = $4,
			purpose = $5,
			scope = $6,
			governance = $7,
			task_config = $8,
			agent_config = $9,
			updated_at = NOW()
		WHERE id = $1
		RETURNING id, key, name, description, status, purpose, scope, governance, task_config, agent_config, created_at, updated_at
	`

	// Marshal JSONB fields
	purposeJSON, err := marshalJSONB(project.Purpose)
	if err != nil {
		return nil, err
	}
	scopeJSON, err := marshalJSONB(project.Scope)
	if err != nil {
		return nil, err
	}
	governanceJSON, err := marshalJSONB(project.Governance)
	if err != nil {
		return nil, err
	}
	taskConfigJSON, err := marshalJSONB(project.TaskConfig)
	if err != nil {
		return nil, err
	}
	agentConfigJSON, err := marshalJSONB(project.AgentConfig)
	if err != nil {
		return nil, err
	}

	var p domain.Project
	var projectID uuid.UUID
	var desc sql.NullString
	var purposeOut, scopeOut, governanceOut, taskConfigOut, agentConfigOut []byte

	err = r.db.QueryRowContext(ctx, query,
		uuid.UUID(project.ID),
		project.Name,
		nullString(project.Description),
		project.Status,
		purposeJSON,
		scopeJSON,
		governanceJSON,
		taskConfigJSON,
		agentConfigJSON,
	).Scan(
		&projectID, &p.Key, &p.Name, &desc, &p.Status, &purposeOut, &scopeOut, &governanceOut, &taskConfigOut, &agentConfigOut, &p.CreatedAt, &p.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	p.ID = domain.ProjectID(projectID)
	if desc.Valid {
		p.Description = desc.String
	}

	// Unmarshal JSONB fields
	if err := unmarshalJSONB(purposeOut, &p.Purpose); err != nil {
		return nil, err
	}
	if err := unmarshalJSONB(scopeOut, &p.Scope); err != nil {
		return nil, err
	}
	if err := unmarshalJSONB(governanceOut, &p.Governance); err != nil {
		return nil, err
	}
	if err := unmarshalJSONB(taskConfigOut, &p.TaskConfig); err != nil {
		return nil, err
	}
	if err := unmarshalJSONB(agentConfigOut, &p.AgentConfig); err != nil {
		return nil, err
	}

	return &p, nil
}

func (r *postgresProjectRepository) Delete(ctx context.Context, id domain.ProjectID) error {
	// First delete project members
	_, err := r.db.ExecContext(ctx, "DELETE FROM aule.project_members WHERE project_id = $1", uuid.UUID(id))
	if err != nil {
		return err
	}

	// Then delete the project
	result, err := r.db.ExecContext(ctx, "DELETE FROM aule.projects WHERE id = $1", uuid.UUID(id))
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return projectsservice.ErrProjectNotFound
	}

	return nil
}

func (r *postgresProjectRepository) IsMember(ctx context.Context, projectID domain.ProjectID, userID domain.UserID) (bool, domain.ProjectMemberRole, error) {
	query := `
		SELECT role FROM aule.project_members
		WHERE project_id = $1 AND user_id = $2
	`

	var role domain.ProjectMemberRole
	err := r.db.QueryRowContext(ctx, query, uuid.UUID(projectID), uuid.UUID(userID)).Scan(&role)
	if errors.Is(err, sql.ErrNoRows) {
		return false, "", nil
	}
	if err != nil {
		return false, "", err
	}

	return true, role, nil
}

func (r *postgresProjectRepository) AddMember(ctx context.Context, projectID domain.ProjectID, userID domain.UserID, role domain.ProjectMemberRole) error {
	query := `
		INSERT INTO aule.project_members (project_id, user_id, role)
		VALUES ($1, $2, $3)
	`

	_, err := r.db.ExecContext(ctx, query, uuid.UUID(projectID), uuid.UUID(userID), role)
	return err
}

func (r *postgresProjectRepository) UpdateMember(ctx context.Context, projectID domain.ProjectID, userID domain.UserID, role domain.ProjectMemberRole, permissions *domain.ProjectMemberPermissions) error {
	permissionsJSON, err := marshalJSONB(permissions)
	if err != nil {
		return err
	}

	query := `
		UPDATE aule.project_members
		SET role = $3, permissions = $4
		WHERE project_id = $1 AND user_id = $2
	`

	result, err := r.db.ExecContext(ctx, query, uuid.UUID(projectID), uuid.UUID(userID), role, permissionsJSON)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return projectsservice.ErrMemberNotFound
	}

	return nil
}

func (r *postgresProjectRepository) RemoveMember(ctx context.Context, projectID domain.ProjectID, userID domain.UserID) error {
	query := `
		DELETE FROM aule.project_members
		WHERE project_id = $1 AND user_id = $2
	`

	result, err := r.db.ExecContext(ctx, query, uuid.UUID(projectID), uuid.UUID(userID))
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return projectsservice.ErrMemberNotFound
	}

	return nil
}

func (r *postgresProjectRepository) ListMembers(ctx context.Context, projectID domain.ProjectID) ([]domain.ProjectMember, error) {
	query := `
		SELECT id, project_id, user_id, role, permissions, created_at
		FROM aule.project_members
		WHERE project_id = $1
		ORDER BY created_at ASC
	`

	rows, err := r.db.QueryContext(ctx, query, uuid.UUID(projectID))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var members []domain.ProjectMember
	for rows.Next() {
		var m domain.ProjectMember
		var memberID, memberProjectID, memberUserID uuid.UUID
		var permissionsJSON []byte

		err := rows.Scan(&memberID, &memberProjectID, &memberUserID, &m.Role, &permissionsJSON, &m.CreatedAt)
		if err != nil {
			return nil, err
		}

		m.ID = memberID
		m.ProjectID = domain.ProjectID(memberProjectID)
		m.UserID = domain.UserID(memberUserID)
		if err := unmarshalJSONB(permissionsJSON, &m.Permissions); err != nil {
			return nil, err
		}

		members = append(members, m)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return members, nil
}

// Helper functions for JSONB marshaling/unmarshaling

func marshalJSONB(v any) ([]byte, error) {
	if v == nil {
		return nil, nil
	}
	return json.Marshal(v)
}

func unmarshalJSONB[T any](data []byte, v *T) error {
	if data == nil || len(data) == 0 {
		return nil
	}
	return json.Unmarshal(data, v)
}

func nullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}

// Ensure pq is imported for array handling (used by wiki spaces)
var _ = pq.Array
