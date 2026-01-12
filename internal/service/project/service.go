package projectsservice

import (
	"context"
	"errors"

	"github.com/dezemandje/aule/internal/domain"
	"github.com/dezemandje/aule/internal/event"
	eventsprojects "github.com/dezemandje/aule/internal/model/events/projects"
)

var (
	ErrProjectNotFound = errors.New("project not found")
	ErrNotAuthorized   = errors.New("not authorized")
	ErrMemberNotFound  = errors.New("member not found")
)

// Service handles project business logic.
type Service struct {
	bus        *event.Bus
	repository Repository
}

// NewService creates a new project service.
func NewService(bus *event.Bus, repository Repository) *Service {
	return &Service{
		bus:        bus,
		repository: repository,
	}
}

// CreateProject creates a new project and publishes a ProjectCreatedEvent.
func (s *Service) CreateProject(ctx context.Context, userID domain.UserID, name, description string) (*domain.Project, error) {
	project := &domain.Project{
		Name:        name,
		Description: description,
		Status:      domain.ProjectStatusActive,
	}

	id, err := s.repository.Create(ctx, project)
	if err != nil {
		return nil, err
	}

	// Add creator as owner
	if err := s.repository.AddMember(ctx, id, userID, domain.ProjectMemberRoleOwner); err != nil {
		return nil, err
	}

	createdProject, err := s.repository.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Publish event
	event.Publish(s.bus, eventsprojects.TopicProjectCreated.Event(eventsprojects.ProjectCreatedEvent{
		ProjectID: createdProject.ID,
		CreatorID: userID,
		Project:   *createdProject,
	}))

	return createdProject, nil
}

// GetProject returns a project by ID if the user has access.
func (s *Service) GetProject(ctx context.Context, userID domain.UserID, projectID domain.ProjectID) (*domain.Project, error) {
	// Check membership
	isMember, _, err := s.repository.IsMember(ctx, projectID, userID)
	if err != nil {
		return nil, err
	}
	if !isMember {
		return nil, ErrNotAuthorized
	}

	project, err := s.repository.FindByID(ctx, projectID)
	if err != nil {
		return nil, err
	}
	if project == nil {
		return nil, ErrProjectNotFound
	}

	return project, nil
}

// ListProjects returns all projects accessible by the user.
func (s *Service) ListProjects(ctx context.Context, userID domain.UserID) ([]domain.Project, error) {
	projects, _, err := s.repository.FindByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	return projects, nil
}

// UpdateProject updates a project and publishes a ProjectUpdatedEvent.
func (s *Service) UpdateProject(ctx context.Context, userID domain.UserID, projectID domain.ProjectID, update *ProjectUpdate) (*domain.Project, error) {
	// Check membership and authorization
	isMember, role, err := s.repository.IsMember(ctx, projectID, userID)
	if err != nil {
		return nil, err
	}
	if !isMember {
		return nil, ErrNotAuthorized
	}
	if !role.CanEdit() {
		return nil, ErrNotAuthorized
	}

	// Fetch current project
	current, err := s.repository.FindByID(ctx, projectID)
	if err != nil {
		return nil, err
	}
	if current == nil {
		return nil, ErrProjectNotFound
	}

	// Apply updates
	if update.Name != nil {
		current.Name = *update.Name
	}
	if update.Description != nil {
		current.Description = *update.Description
	}
	if update.Status != nil {
		current.Status = *update.Status
	}
	if update.Purpose != nil {
		current.Purpose = update.Purpose
	}
	if update.Scope != nil {
		current.Scope = update.Scope
	}
	if update.Governance != nil {
		current.Governance = update.Governance
	}
	if update.TaskConfig != nil {
		current.TaskConfig = update.TaskConfig
	}
	if update.AgentConfig != nil {
		current.AgentConfig = update.AgentConfig
	}

	project, err := s.repository.Update(ctx, current)
	if err != nil {
		return nil, err
	}
	if project == nil {
		return nil, ErrProjectNotFound
	}

	// Publish event
	event.Publish(s.bus, eventsprojects.TopicProjectUpdated.Event(eventsprojects.ProjectUpdatedEvent{
		ProjectID: project.ID,
		UpdaterID: userID,
		Project:   *project,
	}))

	return project, nil
}

// DeleteProject deletes a project and publishes a ProjectDeletedEvent.
func (s *Service) DeleteProject(ctx context.Context, userID domain.UserID, projectID domain.ProjectID) error {
	// Check membership (only owners can delete)
	isMember, role, err := s.repository.IsMember(ctx, projectID, userID)
	if err != nil {
		return err
	}
	if !isMember || !role.CanDelete() {
		return ErrNotAuthorized
	}

	if err := s.repository.Delete(ctx, projectID); err != nil {
		return err
	}

	// Publish event
	event.Publish(s.bus, eventsprojects.TopicProjectDeleted.Event(eventsprojects.ProjectDeletedEvent{
		ProjectID: projectID,
		DeleterID: userID,
	}))

	return nil
}

// ListMembers returns all members of a project.
func (s *Service) ListMembers(ctx context.Context, userID domain.UserID, projectID domain.ProjectID) ([]domain.ProjectMember, error) {
	// Check membership
	isMember, _, err := s.repository.IsMember(ctx, projectID, userID)
	if err != nil {
		return nil, err
	}
	if !isMember {
		return nil, ErrNotAuthorized
	}

	return s.repository.ListMembers(ctx, projectID)
}

// AddMember adds a member to a project.
func (s *Service) AddMember(ctx context.Context, userID domain.UserID, projectID domain.ProjectID, memberUserID domain.UserID, role domain.ProjectMemberRole) error {
	// Check membership and authorization
	isMember, callerRole, err := s.repository.IsMember(ctx, projectID, userID)
	if err != nil {
		return err
	}
	if !isMember || !callerRole.CanManageMembers() {
		return ErrNotAuthorized
	}

	if err := s.repository.AddMember(ctx, projectID, memberUserID, role); err != nil {
		return err
	}

	// Publish event
	event.Publish(s.bus, eventsprojects.TopicMemberAdded.Event(eventsprojects.MemberAddedEvent{
		ProjectID:    projectID,
		MemberUserID: memberUserID,
		Role:         role,
		AddedBy:      userID,
	}))

	return nil
}

// UpdateMember updates a member's role or permissions.
func (s *Service) UpdateMember(ctx context.Context, userID domain.UserID, projectID domain.ProjectID, memberUserID domain.UserID, role domain.ProjectMemberRole, permissions *domain.ProjectMemberPermissions) error {
	// Check membership and authorization
	isMember, callerRole, err := s.repository.IsMember(ctx, projectID, userID)
	if err != nil {
		return err
	}
	if !isMember || !callerRole.CanManageMembers() {
		return ErrNotAuthorized
	}

	if err := s.repository.UpdateMember(ctx, projectID, memberUserID, role, permissions); err != nil {
		return err
	}

	// Publish event
	event.Publish(s.bus, eventsprojects.TopicMemberUpdated.Event(eventsprojects.MemberUpdatedEvent{
		ProjectID:    projectID,
		MemberUserID: memberUserID,
		Role:         role,
		UpdatedBy:    userID,
	}))

	return nil
}

// RemoveMember removes a member from a project.
func (s *Service) RemoveMember(ctx context.Context, userID domain.UserID, projectID domain.ProjectID, memberUserID domain.UserID) error {
	// Check membership and authorization
	isMember, callerRole, err := s.repository.IsMember(ctx, projectID, userID)
	if err != nil {
		return err
	}
	if !isMember || !callerRole.CanManageMembers() {
		return ErrNotAuthorized
	}

	// Prevent removing the last owner
	if callerRole == domain.ProjectMemberRoleOwner {
		members, err := s.repository.ListMembers(ctx, projectID)
		if err != nil {
			return err
		}
		ownerCount := 0
		for _, m := range members {
			if m.Role == domain.ProjectMemberRoleOwner {
				ownerCount++
			}
		}
		// Check if the member being removed is an owner
		_, memberRole, _ := s.repository.IsMember(ctx, projectID, memberUserID)
		if memberRole == domain.ProjectMemberRoleOwner && ownerCount == 1 {
			return errors.New("cannot remove the last owner")
		}
	}

	if err := s.repository.RemoveMember(ctx, projectID, memberUserID); err != nil {
		return err
	}

	// Publish event
	event.Publish(s.bus, eventsprojects.TopicMemberRemoved.Event(eventsprojects.MemberRemovedEvent{
		ProjectID:    projectID,
		MemberUserID: memberUserID,
		RemovedBy:    userID,
	}))

	return nil
}

// ProjectUpdate represents the fields that can be updated on a project.
type ProjectUpdate struct {
	Name        *string                    `json:"name,omitempty"`
	Description *string                    `json:"description,omitempty"`
	Status      *domain.ProjectStatus      `json:"status,omitempty"`
	Purpose     *domain.ProjectPurpose     `json:"purpose,omitempty"`
	Scope       *domain.ProjectScope       `json:"scope,omitempty"`
	Governance  *domain.ProjectGovernance  `json:"governance,omitempty"`
	TaskConfig  *domain.ProjectTaskConfig  `json:"task_config,omitempty"`
	AgentConfig *domain.ProjectAgentConfig `json:"agent_config,omitempty"`
}
