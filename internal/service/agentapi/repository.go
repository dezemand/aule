package agentapi

import (
	"context"
	"time"

	"github.com/dezemandje/aule/internal/domain"
)

// TaskRepository handles task persistence
type TaskRepository interface {
	// FindByID retrieves a task by ID
	FindByID(ctx context.Context, id domain.TaskID) (*domain.Task, error)

	// UpdateStatus updates task status and execution fields
	UpdateStatus(ctx context.Context, id domain.TaskID, status string, claimedBy string, leaseUntil *time.Time) error

	// SetResult sets the task result on completion
	SetResult(ctx context.Context, id domain.TaskID, status string, result string) error

	// SetError sets the task error on failure
	SetError(ctx context.Context, id domain.TaskID, status string, errorMsg string) error
}

// AgentInstanceRepository handles agent instance persistence
type AgentInstanceRepository interface {
	// Create creates a new agent instance
	Create(ctx context.Context, instance *domain.AgentInstance) (domain.AgentInstanceID, error)

	// FindByID retrieves an agent instance by ID
	FindByID(ctx context.Context, id domain.AgentInstanceID) (*domain.AgentInstance, error)

	// UpdateStatus updates instance status
	UpdateStatus(ctx context.Context, id domain.AgentInstanceID, status domain.AgentStatus) error

	// SetCompleted marks instance as completed with result
	SetCompleted(ctx context.Context, id domain.AgentInstanceID, result string, inputTokens, outputTokens int) error

	// SetFailed marks instance as failed with error
	SetFailed(ctx context.Context, id domain.AgentInstanceID, errorMsg string) error
}

// AgentLogRepository handles agent execution logs
type AgentLogRepository interface {
	// Create creates a new log entry
	Create(ctx context.Context, log *AgentLog) error

	// FindByInstanceID retrieves all logs for an instance
	FindByInstanceID(ctx context.Context, instanceID domain.AgentInstanceID) ([]AgentLog, error)
}

// AgentLog represents an execution log entry
type AgentLog struct {
	ID              string
	AgentInstanceID domain.AgentInstanceID
	LogType         string // "text", "tool_call", "tool_result", "error"
	Content         string
	ToolName        string
	ToolInput       string // JSON
	ToolOutput      string
	CreatedAt       time.Time
}
