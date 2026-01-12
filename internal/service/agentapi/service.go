package agentapi

import (
	"context"
	"fmt"
	"time"

	"github.com/dezemandje/aule/internal/domain"
	"github.com/dezemandje/aule/internal/event"
	"github.com/google/uuid"
)

const (
	// Default lease duration for task execution
	DefaultLeaseDuration = 30 * time.Minute

	// Task statuses
	TaskStatusReady     = "ready"
	TaskStatusRunning   = "running"
	TaskStatusDone      = "done"
	TaskStatusFailed    = "failed"
	TaskStatusCancelled = "cancelled"
)

// Service handles agent API operations
type Service struct {
	bus            *event.Bus
	taskRepo       TaskRepository
	agentRepo      AgentInstanceRepository
	logRepo        AgentLogRepository
	defaultWorkDir string
	llmConfig      *LLMConfig // Default LLM configuration
}

// ServiceConfig holds configuration for the agent API service
type ServiceConfig struct {
	DefaultWorkDir   string
	LLMProxyEndpoint string
	LLMProvider      string
	LLMModel         string
	LLMMaxTokens     int
	LLMTemperature   float64
}

// NewService creates a new agent API service
func NewService(
	bus *event.Bus,
	taskRepo TaskRepository,
	agentRepo AgentInstanceRepository,
	logRepo AgentLogRepository,
	cfg ServiceConfig,
) *Service {
	var llmConfig *LLMConfig
	if cfg.LLMProxyEndpoint != "" {
		llmConfig = &LLMConfig{
			Endpoint:    cfg.LLMProxyEndpoint,
			Provider:    cfg.LLMProvider,
			Model:       cfg.LLMModel,
			MaxTokens:   cfg.LLMMaxTokens,
			Temperature: cfg.LLMTemperature,
		}
		// Set defaults
		if llmConfig.Provider == "" {
			llmConfig.Provider = "openai"
		}
		if llmConfig.Model == "" {
			llmConfig.Model = "gpt-4o"
		}
		if llmConfig.MaxTokens == 0 {
			llmConfig.MaxTokens = 4096
		}
	}

	return &Service{
		bus:            bus,
		taskRepo:       taskRepo,
		agentRepo:      agentRepo,
		logRepo:        logRepo,
		defaultWorkDir: cfg.DefaultWorkDir,
		llmConfig:      llmConfig,
	}
}

// GetTaskDetails retrieves task details for agent execution
func (s *Service) GetTaskDetails(ctx context.Context, taskID domain.TaskID) (*TaskDetailsResponse, error) {
	task, err := s.taskRepo.FindByID(ctx, taskID)
	if err != nil {
		return nil, fmt.Errorf("task not found: %w", err)
	}

	// Build allowed tools list - default to basic set if not specified
	allowedTools := task.AllowedTools
	if len(allowedTools) == 0 {
		allowedTools = []string{"read", "write", "edit", "glob", "grep", "bash"}
	}

	// Build system prompt
	systemPrompt := task.SystemPrompt
	if systemPrompt == "" {
		systemPrompt = buildDefaultSystemPrompt(task)
	}

	return &TaskDetailsResponse{
		Task:         TaskToInfo(task),
		SystemPrompt: systemPrompt,
		Context:      task.Context,
		AllowedTools: allowedTools,
		WorkDir:      s.defaultWorkDir,
		LLMConfig:    s.llmConfig,
	}, nil
}

// StartTask marks a task as running and creates an agent instance
func (s *Service) StartTask(ctx context.Context, taskID domain.TaskID, agentID string) (*TaskStartResponse, error) {
	task, err := s.taskRepo.FindByID(ctx, taskID)
	if err != nil {
		return nil, fmt.Errorf("task not found: %w", err)
	}

	// Verify task is in ready state
	if task.Status != TaskStatusReady {
		return nil, fmt.Errorf("task is not ready (status: %s)", task.Status)
	}

	// Set lease expiration
	leaseUntil := time.Now().Add(DefaultLeaseDuration)

	// Update task status
	err = s.taskRepo.UpdateStatus(ctx, taskID, TaskStatusRunning, agentID, &leaseUntil)
	if err != nil {
		return nil, fmt.Errorf("failed to update task status: %w", err)
	}

	// Create agent instance
	agentTypeID := task.AgentType
	if agentTypeID == uuid.Nil {
		// Use a default agent type based on task type
		// For now, just use a nil UUID - in production, look up by name
		agentTypeID = uuid.Nil
	}

	instance := &domain.AgentInstance{
		ProjectID: domain.ProjectID(task.ProjectID),
		AgentType: domain.AgentTypeID(agentTypeID),
		TaskID:    taskID,
		Status:    domain.AgentStatusRunning,
	}

	instanceID, err := s.agentRepo.Create(ctx, instance)
	if err != nil {
		// Rollback task status
		_ = s.taskRepo.UpdateStatus(ctx, taskID, TaskStatusReady, "", nil)
		return nil, fmt.Errorf("failed to create agent instance: %w", err)
	}

	return &TaskStartResponse{
		SessionID:   uuid.New().String(),
		InstanceID:  uuid.UUID(instanceID).String(),
		WorkDir:     s.defaultWorkDir,
		MaxTokens:   4096,
		Model:       "gpt-4o", // Default, can be overridden
		Temperature: 0.0,
	}, nil
}

// UpdateTask records progress during task execution
func (s *Service) UpdateTask(ctx context.Context, taskID domain.TaskID, instanceID domain.AgentInstanceID, req *TaskUpdateRequest) error {
	// Log text content
	if req.Content != "" {
		err := s.logRepo.Create(ctx, &AgentLog{
			AgentInstanceID: instanceID,
			LogType:         "text",
			Content:         req.Content,
			CreatedAt:       time.Now(),
		})
		if err != nil {
			return fmt.Errorf("failed to log content: %w", err)
		}
	}

	// Log tool calls
	for _, tc := range req.ToolCalls {
		logType := "tool_call"
		if tc.Error != "" {
			logType = "tool_error"
		}

		err := s.logRepo.Create(ctx, &AgentLog{
			AgentInstanceID: instanceID,
			LogType:         logType,
			ToolName:        tc.Name,
			ToolInput:       tc.Input,
			ToolOutput:      tc.Output,
			Content:         tc.Error,
			CreatedAt:       time.Now(),
		})
		if err != nil {
			return fmt.Errorf("failed to log tool call: %w", err)
		}
	}

	// Extend lease if task is still running
	leaseUntil := time.Now().Add(DefaultLeaseDuration)
	task, err := s.taskRepo.FindByID(ctx, taskID)
	if err == nil && task.Status == TaskStatusRunning {
		_ = s.taskRepo.UpdateStatus(ctx, taskID, TaskStatusRunning, task.ClaimedBy, &leaseUntil)
	}

	return nil
}

// CompleteTask marks a task as successfully completed
func (s *Service) CompleteTask(ctx context.Context, taskID domain.TaskID, instanceID domain.AgentInstanceID, req *TaskCompleteRequest) error {
	// Update task status
	err := s.taskRepo.SetResult(ctx, taskID, TaskStatusDone, req.Result)
	if err != nil {
		return fmt.Errorf("failed to complete task: %w", err)
	}

	// Update agent instance
	err = s.agentRepo.SetCompleted(ctx, instanceID, req.Result, req.TokensUsed.InputTokens, req.TokensUsed.OutputTokens)
	if err != nil {
		return fmt.Errorf("failed to update agent instance: %w", err)
	}

	return nil
}

// FailTask marks a task as failed
func (s *Service) FailTask(ctx context.Context, taskID domain.TaskID, instanceID domain.AgentInstanceID, req *TaskFailRequest) error {
	// Update task status
	err := s.taskRepo.SetError(ctx, taskID, TaskStatusFailed, req.Error)
	if err != nil {
		return fmt.Errorf("failed to fail task: %w", err)
	}

	// Update agent instance
	err = s.agentRepo.SetFailed(ctx, instanceID, req.Error)
	if err != nil {
		return fmt.Errorf("failed to update agent instance: %w", err)
	}

	return nil
}

// buildDefaultSystemPrompt creates a default system prompt for a task
func buildDefaultSystemPrompt(task *domain.Task) string {
	return fmt.Sprintf(`You are an AI agent working on a software development task.

Task Type: %s
Task Title: %s

You have access to tools to help you complete this task. Use them wisely:
- read: Read file contents
- write: Write/create files
- edit: Edit files with string replacement
- glob: Find files by pattern
- grep: Search file contents
- bash: Execute shell commands

Guidelines:
1. Start by understanding the task and exploring relevant code
2. Make small, focused changes
3. Test your changes when possible
4. Explain your reasoning as you work

Complete the task thoroughly and report your results.`, task.Type, task.Title)
}
