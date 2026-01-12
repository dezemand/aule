// Package modelsws defines WebSocket message types.
package modelsws

import "github.com/dezemandje/aule/internal/domain"

// Message type constants for project operations.
const (
	MsgTypeProjectsListReq = "projects.list.req"
	MsgTypeProjectsList    = "projects.list"
	MsgTypeProjectCreate   = "projects.create.req"
	MsgTypeProjectCreated  = "projects.created"
	MsgTypeProjectUpdate   = "projects.update.req"
	MsgTypeProjectUpdated  = "projects.updated"
	MsgTypeProjectDelete   = "projects.delete.req"
	MsgTypeProjectDeleted  = "projects.deleted"
	MsgTypeProjectGet      = "projects.get.req"
	MsgTypeProject         = "projects.get"

	// Member management
	MsgTypeMembersListReq = "projects.members.list.req"
	MsgTypeMembersList    = "projects.members.list"
	MsgTypeMemberAdd      = "projects.members.add.req"
	MsgTypeMemberAdded    = "projects.members.added"
	MsgTypeMemberUpdate   = "projects.members.update.req"
	MsgTypeMemberUpdated  = "projects.members.updated"
	MsgTypeMemberRemove   = "projects.members.remove.req"
	MsgTypeMemberRemoved  = "projects.members.removed"

	// Repository management
	MsgTypeReposListReq = "projects.repos.list.req"
	MsgTypeReposList    = "projects.repos.list"
	MsgTypeRepoAdd      = "projects.repos.add.req"
	MsgTypeRepoAdded    = "projects.repos.added"
	MsgTypeRepoUpdate   = "projects.repos.update.req"
	MsgTypeRepoUpdated  = "projects.repos.updated"
	MsgTypeRepoRemove   = "projects.repos.remove.req"
	MsgTypeRepoRemoved  = "projects.repos.removed"
)

// ProjectsListRequest requests a list of projects for the current user.
type ProjectsListRequest struct{}

func (p *ProjectsListRequest) Type() string { return MsgTypeProjectsListReq }

// ProjectsListResponse contains the list of projects.
type ProjectsListResponse struct {
	Projects []domain.Project `json:"projects"`
}

func (p *ProjectsListResponse) Type() string { return MsgTypeProjectsList }

// ProjectCreateRequest creates a new project.
type ProjectCreateRequest struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

func (p *ProjectCreateRequest) Type() string { return MsgTypeProjectCreate }

// ProjectCreatedResponse is returned after creating a project.
type ProjectCreatedResponse struct {
	Project domain.Project `json:"project"`
}

func (p *ProjectCreatedResponse) Type() string { return MsgTypeProjectCreated }

// ProjectUpdateRequest updates an existing project.
type ProjectUpdateRequest struct {
	ID          string                     `json:"id"`
	Name        *string                    `json:"name,omitempty"`
	Description *string                    `json:"description,omitempty"`
	Status      *domain.ProjectStatus      `json:"status,omitempty"`
	Purpose     *domain.ProjectPurpose     `json:"purpose,omitempty"`
	Scope       *domain.ProjectScope       `json:"scope,omitempty"`
	Governance  *domain.ProjectGovernance  `json:"governance,omitempty"`
	TaskConfig  *domain.ProjectTaskConfig  `json:"task_config,omitempty"`
	AgentConfig *domain.ProjectAgentConfig `json:"agent_config,omitempty"`
}

func (p *ProjectUpdateRequest) Type() string { return MsgTypeProjectUpdate }

// ProjectUpdatedResponse is returned after updating a project.
type ProjectUpdatedResponse struct {
	Project domain.Project `json:"project"`
}

func (p *ProjectUpdatedResponse) Type() string { return MsgTypeProjectUpdated }

// ProjectDeleteRequest deletes a project.
type ProjectDeleteRequest struct {
	ID string `json:"id"`
}

func (p *ProjectDeleteRequest) Type() string { return MsgTypeProjectDelete }

// ProjectDeletedResponse is returned after deleting a project.
type ProjectDeletedResponse struct {
	ID string `json:"id"`
}

func (p *ProjectDeletedResponse) Type() string { return MsgTypeProjectDeleted }

// ProjectGetRequest requests a single project by ID.
type ProjectGetRequest struct {
	ID string `json:"id"`
}

func (p *ProjectGetRequest) Type() string { return MsgTypeProjectGet }

// ProjectResponse returns a single project.
type ProjectResponse struct {
	Project domain.Project `json:"project"`
}

func (p *ProjectResponse) Type() string { return MsgTypeProject }

// --- Member Management Messages ---

// MembersListRequest requests the list of members for a project.
type MembersListRequest struct {
	ProjectID string `json:"project_id"`
}

func (m *MembersListRequest) Type() string { return MsgTypeMembersListReq }

// MembersListResponse contains the list of project members.
type MembersListResponse struct {
	Members []domain.ProjectMember `json:"members"`
}

func (m *MembersListResponse) Type() string { return MsgTypeMembersList }

// MemberAddRequest adds a member to a project.
type MemberAddRequest struct {
	ProjectID   string                           `json:"project_id"`
	UserID      string                           `json:"user_id"`
	Role        domain.ProjectMemberRole         `json:"role"`
	Permissions *domain.ProjectMemberPermissions `json:"permissions,omitempty"`
}

func (m *MemberAddRequest) Type() string { return MsgTypeMemberAdd }

// MemberAddedResponse is returned after adding a member.
type MemberAddedResponse struct {
	ProjectID string `json:"project_id"`
	UserID    string `json:"user_id"`
}

func (m *MemberAddedResponse) Type() string { return MsgTypeMemberAdded }

// MemberUpdateRequest updates a member's role or permissions.
type MemberUpdateRequest struct {
	ProjectID   string                           `json:"project_id"`
	UserID      string                           `json:"user_id"`
	Role        domain.ProjectMemberRole         `json:"role"`
	Permissions *domain.ProjectMemberPermissions `json:"permissions,omitempty"`
}

func (m *MemberUpdateRequest) Type() string { return MsgTypeMemberUpdate }

// MemberUpdatedResponse is returned after updating a member.
type MemberUpdatedResponse struct {
	ProjectID string `json:"project_id"`
	UserID    string `json:"user_id"`
}

func (m *MemberUpdatedResponse) Type() string { return MsgTypeMemberUpdated }

// MemberRemoveRequest removes a member from a project.
type MemberRemoveRequest struct {
	ProjectID string `json:"project_id"`
	UserID    string `json:"user_id"`
}

func (m *MemberRemoveRequest) Type() string { return MsgTypeMemberRemove }

// MemberRemovedResponse is returned after removing a member.
type MemberRemovedResponse struct {
	ProjectID string `json:"project_id"`
	UserID    string `json:"user_id"`
}

func (m *MemberRemovedResponse) Type() string { return MsgTypeMemberRemoved }

// --- Repository Management Messages ---

// ReposListRequest requests the list of repositories for a project.
type ReposListRequest struct {
	ProjectID string `json:"project_id"`
}

func (r *ReposListRequest) Type() string { return MsgTypeReposListReq }

// ReposListResponse contains the list of project repositories.
type ReposListResponse struct {
	Repositories []domain.ProjectRepository `json:"repositories"`
}

func (r *ReposListResponse) Type() string { return MsgTypeReposList }

// RepoAddRequest adds a repository to a project.
type RepoAddRequest struct {
	ProjectID              string               `json:"project_id"`
	URL                    string               `json:"url"`
	Purpose                string               `json:"purpose"`
	DefaultBranch          string               `json:"default_branch,omitempty"`
	AllowedPaths           *domain.AllowedPaths `json:"allowed_paths,omitempty"`
	BranchNamingConvention string               `json:"branch_naming_convention,omitempty"`
}

func (r *RepoAddRequest) Type() string { return MsgTypeRepoAdd }

// RepoAddedResponse is returned after adding a repository.
type RepoAddedResponse struct {
	Repository domain.ProjectRepository `json:"repository"`
}

func (r *RepoAddedResponse) Type() string { return MsgTypeRepoAdded }

// RepoUpdateRequest updates a repository's settings.
type RepoUpdateRequest struct {
	ID                     string               `json:"id"`
	Purpose                *string              `json:"purpose,omitempty"`
	DefaultBranch          *string              `json:"default_branch,omitempty"`
	AllowedPaths           *domain.AllowedPaths `json:"allowed_paths,omitempty"`
	BranchNamingConvention *string              `json:"branch_naming_convention,omitempty"`
}

func (r *RepoUpdateRequest) Type() string { return MsgTypeRepoUpdate }

// RepoUpdatedResponse is returned after updating a repository.
type RepoUpdatedResponse struct {
	Repository domain.ProjectRepository `json:"repository"`
}

func (r *RepoUpdatedResponse) Type() string { return MsgTypeRepoUpdated }

// RepoRemoveRequest removes a repository from a project.
type RepoRemoveRequest struct {
	ID string `json:"id"`
}

func (r *RepoRemoveRequest) Type() string { return MsgTypeRepoRemove }

// RepoRemovedResponse is returned after removing a repository.
type RepoRemovedResponse struct {
	ID string `json:"id"`
}

func (r *RepoRemovedResponse) Type() string { return MsgTypeRepoRemoved }
