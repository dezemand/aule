package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type ProjectID uuid.UUID

// ProjectStatus represents the lifecycle state of a project
type ProjectStatus string

const (
	ProjectStatusActive   ProjectStatus = "active"
	ProjectStatusPaused   ProjectStatus = "paused"
	ProjectStatusArchived ProjectStatus = "archived"
)

// ProjectPurpose defines why the project exists (Section 2)
type ProjectPurpose struct {
	Goal             string   `json:"goal,omitempty"`
	ProblemStatement string   `json:"problem_statement,omitempty"`
	NonGoals         []string `json:"non_goals,omitempty"`
	ExpectedValue    string   `json:"expected_value,omitempty"`
	TimeHorizon      string   `json:"time_horizon,omitempty"`
}

// ProjectScope defines the boundaries of the project (Section 3)
type ProjectScope struct {
	InScope     []string `json:"in_scope,omitempty"`
	OutOfScope  []string `json:"out_of_scope,omitempty"`
	Assumptions []string `json:"assumptions,omitempty"`
	Constraints []string `json:"constraints,omitempty"`
}

// ProjectGovernance defines autonomy and control settings (Section 5)
type ProjectGovernance struct {
	AutonomyLevel     string   `json:"autonomy_level,omitempty"`     // assistive | supervised | autonomous | overnight-autonomous
	HumanInTheLoop    []string `json:"human_in_the_loop,omitempty"`  // stages requiring approval
	ReviewStrictness  string   `json:"review_strictness,omitempty"`  // light | normal | strict
	DecisionAuthority string   `json:"decision_authority,omitempty"` // propose | decide
	EscalationRules   []string `json:"escalation_rules,omitempty"`
}

// ProjectTaskConfig defines task model settings (Section 6)
type ProjectTaskConfig struct {
	AllowedTaskTypes     []string          `json:"allowed_task_types,omitempty"`
	CustomStageOverrides map[string]any    `json:"custom_stage_overrides,omitempty"`
	DefaultPriorities    map[string]string `json:"default_priorities,omitempty"`
	WIPLimits            map[string]int    `json:"wip_limits,omitempty"`
}

// ProjectAgentConfig defines agent execution settings (Section 7)
type ProjectAgentConfig struct {
	AllowedAgentTypes  []string       `json:"allowed_agent_types,omitempty"`
	TrustLevel         string         `json:"trust_level,omitempty"` // experimental | operational | trusted
	RuntimePermissions []string       `json:"runtime_permissions,omitempty"`
	MaxParallelAgents  int            `json:"max_parallel_agents,omitempty"`
	BudgetLimits       map[string]any `json:"budget_limits,omitempty"`
}

// Project is the top-level context boundary for work
type Project struct {
	ID          ProjectID     `json:"id"`
	Key         string        `json:"key"`
	Name        string        `json:"name"`
	Description string        `json:"description,omitempty"`
	Status      ProjectStatus `json:"status"`

	// Structured configuration sections
	Purpose     *ProjectPurpose     `json:"purpose,omitempty"`
	Scope       *ProjectScope       `json:"scope,omitempty"`
	Governance  *ProjectGovernance  `json:"governance,omitempty"`
	TaskConfig  *ProjectTaskConfig  `json:"task_config,omitempty"`
	AgentConfig *ProjectAgentConfig `json:"agent_config,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ProjectMemberRole defines the access level of a project member
type ProjectMemberRole string

const (
	ProjectMemberRoleOwner       ProjectMemberRole = "owner"
	ProjectMemberRoleAdmin       ProjectMemberRole = "admin"
	ProjectMemberRoleContributor ProjectMemberRole = "contributor"
	ProjectMemberRoleReviewer    ProjectMemberRole = "reviewer"
	ProjectMemberRoleViewer      ProjectMemberRole = "viewer"
)

// ProjectMemberPermissions defines granular permissions for a member
type ProjectMemberPermissions struct {
	CanEditProject    bool `json:"can_edit_project,omitempty"`
	CanManageMembers  bool `json:"can_manage_members,omitempty"`
	CanCreateTasks    bool `json:"can_create_tasks,omitempty"`
	CanRunAgents      bool `json:"can_run_agents,omitempty"`
	CanApproveChanges bool `json:"can_approve_changes,omitempty"`
}

// ProjectMember represents a user's membership in a project
type ProjectMember struct {
	ID          uuid.UUID                 `json:"id"`
	ProjectID   ProjectID                 `json:"project_id"`
	UserID      UserID                    `json:"user_id"`
	Role        ProjectMemberRole         `json:"role"`
	Permissions *ProjectMemberPermissions `json:"permissions,omitempty"`
	CreatedAt   time.Time                 `json:"created_at"`
}

// AllowedPaths defines read/write path permissions for a repository
type AllowedPaths struct {
	Read  []string `json:"read,omitempty"`
	Write []string `json:"write,omitempty"`
}

// ProjectRepository represents a git repository attached to a project (Section 8)
type ProjectRepository struct {
	ID                     uuid.UUID     `json:"id"`
	ProjectID              ProjectID     `json:"project_id"`
	URL                    string        `json:"url"`
	Purpose                string        `json:"purpose"` // code | docs | infra | examples
	DefaultBranch          string        `json:"default_branch"`
	AllowedPaths           *AllowedPaths `json:"allowed_paths,omitempty"`
	BranchNamingConvention string        `json:"branch_naming_convention,omitempty"`
	CreatedAt              time.Time     `json:"created_at"`
	UpdatedAt              time.Time     `json:"updated_at"`
}

// ProjectWikiSpace represents a wiki space attached to a project (Section 8)
type ProjectWikiSpace struct {
	ID           uuid.UUID `json:"id"`
	ProjectID    ProjectID `json:"project_id"`
	SpaceID      string    `json:"space_id"`
	AccessMode   string    `json:"access_mode"` // read | write
	PagePrefixes []string  `json:"page_prefixes,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// ProjectID JSON marshaling

func (id ProjectID) MarshalJSON() ([]byte, error) {
	u := uuid.UUID(id)
	return json.Marshal(u.String())
}

func (id *ProjectID) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}

	u, err := uuid.Parse(s)
	if err != nil {
		return err
	}

	*id = ProjectID(u)
	return nil
}

func (id ProjectID) String() string {
	u := uuid.UUID(id)
	return u.String()
}

// Helper methods for role-based authorization

// CanEdit returns true if the role allows editing project settings
func (r ProjectMemberRole) CanEdit() bool {
	return r == ProjectMemberRoleOwner || r == ProjectMemberRoleAdmin
}

// CanManageMembers returns true if the role allows managing project members
func (r ProjectMemberRole) CanManageMembers() bool {
	return r == ProjectMemberRoleOwner || r == ProjectMemberRoleAdmin
}

// CanDelete returns true if the role allows deleting the project
func (r ProjectMemberRole) CanDelete() bool {
	return r == ProjectMemberRoleOwner
}

// CanCreateTasks returns true if the role allows creating tasks
func (r ProjectMemberRole) CanCreateTasks() bool {
	return r == ProjectMemberRoleOwner || r == ProjectMemberRoleAdmin || r == ProjectMemberRoleContributor
}
