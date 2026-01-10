package dbmemory

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/dezemandje/aule/internal/domain"
	"github.com/google/uuid"
)

// TaskRepository is an in-memory implementation of TaskRepository
type TaskRepository struct {
	mu    sync.RWMutex
	tasks map[domain.TaskID]*domain.Task
}

// NewTaskRepository creates a new in-memory task repository
func NewTaskRepository() *TaskRepository {
	return &TaskRepository{
		tasks: make(map[domain.TaskID]*domain.Task),
	}
}

// Create creates a new task
func (r *TaskRepository) Create(ctx context.Context, task *domain.Task) (domain.TaskID, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if task.ID == domain.TaskID(uuid.Nil) {
		task.ID = domain.TaskID(uuid.New())
	}
	task.CreatedAt = time.Now()
	task.UpdatedAt = time.Now()

	r.tasks[task.ID] = task
	return task.ID, nil
}

// FindByID retrieves a task by ID
func (r *TaskRepository) FindByID(ctx context.Context, id domain.TaskID) (*domain.Task, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	task, ok := r.tasks[id]
	if !ok {
		return nil, fmt.Errorf("task not found: %s", uuid.UUID(id).String())
	}

	// Return a copy to prevent mutation
	taskCopy := *task
	return &taskCopy, nil
}

// UpdateStatus updates task status and execution fields
func (r *TaskRepository) UpdateStatus(ctx context.Context, id domain.TaskID, status string, claimedBy string, leaseUntil *time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	task, ok := r.tasks[id]
	if !ok {
		return fmt.Errorf("task not found: %s", uuid.UUID(id).String())
	}

	task.Status = status
	task.ClaimedBy = claimedBy
	task.LeaseUntil = leaseUntil
	task.UpdatedAt = time.Now()

	return nil
}

// SetResult sets the task result on completion
func (r *TaskRepository) SetResult(ctx context.Context, id domain.TaskID, status string, result string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	task, ok := r.tasks[id]
	if !ok {
		return fmt.Errorf("task not found: %s", uuid.UUID(id).String())
	}

	task.Status = status
	task.ClaimedBy = ""
	task.LeaseUntil = nil
	task.UpdatedAt = time.Now()
	// Note: Result stored in context for now (could add dedicated field)

	return nil
}

// SetError sets the task error on failure
func (r *TaskRepository) SetError(ctx context.Context, id domain.TaskID, status string, errorMsg string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	task, ok := r.tasks[id]
	if !ok {
		return fmt.Errorf("task not found: %s", uuid.UUID(id).String())
	}

	task.Status = status
	task.ClaimedBy = ""
	task.LeaseUntil = nil
	task.UpdatedAt = time.Now()

	return nil
}

// List returns all tasks (for testing)
func (r *TaskRepository) List(ctx context.Context) ([]*domain.Task, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tasks := make([]*domain.Task, 0, len(r.tasks))
	for _, task := range r.tasks {
		taskCopy := *task
		tasks = append(tasks, &taskCopy)
	}
	return tasks, nil
}

// SeedTestTask creates a test task for development
func (r *TaskRepository) SeedTestTask(projectID uuid.UUID) domain.TaskID {
	task := &domain.Task{
		ID:           domain.TaskID(uuid.New()),
		ProjectID:    projectID,
		Title:        "Test Task: Explore the codebase",
		Description:  "Explore the codebase and provide a summary of the project structure and main components.",
		Type:         "exploration",
		Status:       "ready",
		Priority:     1,
		Labels:       []string{"test", "exploration"},
		Context:      "This is a Go project with a backend API and frontend. Focus on understanding the overall architecture.",
		AllowedTools: []string{"read", "glob", "grep", "bash"},
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	r.mu.Lock()
	r.tasks[task.ID] = task
	r.mu.Unlock()

	return task.ID
}
