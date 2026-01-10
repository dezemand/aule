package domain

import (
	"time"

	"github.com/google/uuid"
)

type TaskID uuid.UUID

type Task struct {
	ID        TaskID    `json:"id"`
	ProjectID uuid.UUID `json:"project_id"`

	Title     string    `json:"title"`
	Type      string    `json:"type"`
	Status    string    `json:"status"`
	StageKey  string    `json:"stage_key,omitempty"` // aligns with BoardStage.Key
	Priority  int       `json:"priority,omitempty"`  // higher = sooner
	Labels    []string  `json:"labels,omitempty"`
	Assignee  string    `json:"assignee,omitempty"`   // user or agent (later)
	AgentType uuid.UUID `json:"agent_type,omitempty"` // which agent type should run it

	// Execution control (DB lease model)
	ClaimedBy  string     `json:"claimed_by,omitempty"` // executor id
	LeaseUntil *time.Time `json:"lease_until,omitempty"`
	AttemptID  string     `json:"attempt_id,omitempty"`

	// Agent execution context
	Description  string   `json:"description,omitempty"`
	Context      string   `json:"context,omitempty"`       // Additional context for the agent
	SystemPrompt string   `json:"system_prompt,omitempty"` // Custom system prompt
	AllowedTools []string `json:"allowed_tools,omitempty"` // Tools the agent can use

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
