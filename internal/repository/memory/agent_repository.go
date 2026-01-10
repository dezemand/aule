package dbmemory

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/dezemandje/aule/internal/domain"
	"github.com/dezemandje/aule/internal/service/agentapi"
	"github.com/google/uuid"
)

// AgentInstanceRepository is an in-memory implementation
type AgentInstanceRepository struct {
	mu        sync.RWMutex
	instances map[domain.AgentInstanceID]*domain.AgentInstance
}

// NewAgentInstanceRepository creates a new in-memory agent instance repository
func NewAgentInstanceRepository() *AgentInstanceRepository {
	return &AgentInstanceRepository{
		instances: make(map[domain.AgentInstanceID]*domain.AgentInstance),
	}
}

// Create creates a new agent instance
func (r *AgentInstanceRepository) Create(ctx context.Context, instance *domain.AgentInstance) (domain.AgentInstanceID, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if instance.ID == domain.AgentInstanceID(uuid.Nil) {
		instance.ID = domain.AgentInstanceID(uuid.New())
	}
	instance.CreatedAt = time.Now()
	instance.LastUpdated = time.Now()

	r.instances[instance.ID] = instance
	return instance.ID, nil
}

// FindByID retrieves an agent instance by ID
func (r *AgentInstanceRepository) FindByID(ctx context.Context, id domain.AgentInstanceID) (*domain.AgentInstance, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	instance, ok := r.instances[id]
	if !ok {
		return nil, fmt.Errorf("agent instance not found: %s", uuid.UUID(id).String())
	}

	// Return a copy
	instanceCopy := *instance
	return &instanceCopy, nil
}

// UpdateStatus updates instance status
func (r *AgentInstanceRepository) UpdateStatus(ctx context.Context, id domain.AgentInstanceID, status domain.AgentStatus) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	instance, ok := r.instances[id]
	if !ok {
		return fmt.Errorf("agent instance not found: %s", uuid.UUID(id).String())
	}

	instance.Status = status
	instance.LastUpdated = time.Now()
	return nil
}

// SetCompleted marks instance as completed with result
func (r *AgentInstanceRepository) SetCompleted(ctx context.Context, id domain.AgentInstanceID, result string, inputTokens, outputTokens int) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	instance, ok := r.instances[id]
	if !ok {
		return fmt.Errorf("agent instance not found: %s", uuid.UUID(id).String())
	}

	instance.Status = domain.AgentStatusCompleted
	instance.LastUpdated = time.Now()
	// Note: Result and tokens stored in extended fields (would add to domain model)
	return nil
}

// SetFailed marks instance as failed with error
func (r *AgentInstanceRepository) SetFailed(ctx context.Context, id domain.AgentInstanceID, errorMsg string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	instance, ok := r.instances[id]
	if !ok {
		return fmt.Errorf("agent instance not found: %s", uuid.UUID(id).String())
	}

	instance.Status = domain.AgentStatusFailed
	instance.LastUpdated = time.Now()
	return nil
}

// AgentLogRepository is an in-memory implementation
type AgentLogRepository struct {
	mu   sync.RWMutex
	logs []agentapi.AgentLog
}

// NewAgentLogRepository creates a new in-memory agent log repository
func NewAgentLogRepository() *AgentLogRepository {
	return &AgentLogRepository{
		logs: make([]agentapi.AgentLog, 0),
	}
}

// Create creates a new log entry
func (r *AgentLogRepository) Create(ctx context.Context, log *agentapi.AgentLog) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if log.ID == "" {
		log.ID = uuid.New().String()
	}
	if log.CreatedAt.IsZero() {
		log.CreatedAt = time.Now()
	}

	r.logs = append(r.logs, *log)
	return nil
}

// FindByInstanceID retrieves all logs for an instance
func (r *AgentLogRepository) FindByInstanceID(ctx context.Context, instanceID domain.AgentInstanceID) ([]agentapi.AgentLog, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]agentapi.AgentLog, 0)
	for _, log := range r.logs {
		if log.AgentInstanceID == instanceID {
			result = append(result, log)
		}
	}
	return result, nil
}
