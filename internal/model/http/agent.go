// Package modelshttp defines HTTP request/response types.
package modelshttp

import (
	"time"

	"github.com/dezemandje/aule/internal/domain"
	"github.com/google/uuid"
)

// TaskDetailsResponse is returned when agent fetches task details.
type TaskDetailsResponse struct {
	Task         TaskInfo   `json:"task"`
	SystemPrompt string     `json:"system_prompt"`
	Context      string     `json:"context"`
	AllowedTools []string   `json:"allowed_tools"`
	WorkDir      string     `json:"work_dir"`
	LLMConfig    *LLMConfig `json:"llm_config,omitempty"`
}

// LLMConfig specifies how the agent should connect to the LLM.
type LLMConfig struct {
	Endpoint    string  `json:"endpoint"`    // LLM Proxy endpoint URL
	Provider    string  `json:"provider"`    // Provider name (e.g., "openai")
	Model       string  `json:"model"`       // Model ID (e.g., "gpt-4o")
	MaxTokens   int     `json:"max_tokens"`  // Max output tokens
	Temperature float64 `json:"temperature"` // Temperature for sampling
}

// TaskInfo contains task information for the agent.
type TaskInfo struct {
	ID          string    `json:"id"`
	ProjectID   string    `json:"project_id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Type        string    `json:"type"`
	Status      string    `json:"status"`
	Priority    int       `json:"priority"`
	Labels      []string  `json:"labels"`
	CreatedAt   time.Time `json:"created_at"`
}

// TaskStartRequest is sent when agent starts a task.
type TaskStartRequest struct{}

// TaskStartResponse is returned when task is started.
type TaskStartResponse struct {
	SessionID   string  `json:"session_id"`
	InstanceID  string  `json:"instance_id"`
	WorkDir     string  `json:"work_dir"`
	MaxTokens   int     `json:"max_tokens"`
	Model       string  `json:"model"`
	Temperature float64 `json:"temperature"`
}

// TaskUpdateRequest is sent during task execution.
type TaskUpdateRequest struct {
	Content    string      `json:"content,omitempty"`
	ToolCalls  []ToolCall  `json:"tool_calls,omitempty"`
	TokensUsed *TokenUsage `json:"tokens_used,omitempty"`
}

// ToolCall represents a tool invocation.
type ToolCall struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Input  string `json:"input"` // JSON string
	Output string `json:"output"`
	Error  string `json:"error,omitempty"`
}

// TokenUsage tracks token consumption.
type TokenUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

// TaskCompleteRequest is sent when task completes successfully.
type TaskCompleteRequest struct {
	Result     string     `json:"result"`
	TokensUsed TokenUsage `json:"tokens_used"`
}

// TaskFailRequest is sent when task fails.
type TaskFailRequest struct {
	Error      string     `json:"error"`
	TokensUsed TokenUsage `json:"tokens_used,omitempty"`
}

// APIResponse is a generic API response.
type APIResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
	Error   string `json:"error,omitempty"`
}

// TaskToInfo converts a domain.Task to TaskInfo.
func TaskToInfo(t *domain.Task) TaskInfo {
	return TaskInfo{
		ID:          taskIDToString(t.ID),
		ProjectID:   t.ProjectID.String(),
		Title:       t.Title,
		Description: t.Description,
		Type:        t.Type,
		Status:      t.Status,
		Priority:    t.Priority,
		Labels:      t.Labels,
		CreatedAt:   t.CreatedAt,
	}
}

func taskIDToString(id domain.TaskID) string {
	return uuid.UUID(id).String()
}
