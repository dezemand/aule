package domain

import (
	"time"

	"github.com/google/uuid"
)

type AgentTypeID uuid.UUID

type AgentType struct {
	ID   AgentTypeID
	Name string
}

type AgentToolCapability struct {
	AgentTypeID AgentTypeID
	ToolID      ToolID
}

type AgentInstanceID uuid.UUID

type AgentStatus string

const (
	AgentStatusIdle       AgentStatus = "idle"
	AgentStatusRunning    AgentStatus = "running"
	AgentStatusCompleted  AgentStatus = "completed"
	AgentStatusFailed     AgentStatus = "failed"
	AgentStatusTerminated AgentStatus = "terminated"
)

type AgentInstance struct {
	ID        AgentInstanceID `json:"id"`
	ProjectID ProjectID       `json:"project_id"`
	AgentType AgentTypeID     `json:"agent_type_id"`
	TaskID    TaskID          `json:"task_id,omitempty"`

	CreatedAt   time.Time `json:"created_at"`
	LastUpdated time.Time `json:"last_updated"`

	Status AgentStatus `json:"status"`
}
