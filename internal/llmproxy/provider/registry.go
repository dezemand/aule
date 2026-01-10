package provider

import (
	"fmt"
	"sync"
)

// Registry manages available LLM providers
type Registry struct {
	mu        sync.RWMutex
	providers map[string]Provider
}

// NewRegistry creates a new provider registry
func NewRegistry() *Registry {
	return &Registry{
		providers: make(map[string]Provider),
	}
}

// Register adds a provider to the registry
func (r *Registry) Register(p Provider) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.providers[p.Name()] = p
}

// Get retrieves a provider by name
func (r *Registry) Get(name string) (Provider, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	p, ok := r.providers[name]
	if !ok {
		return nil, fmt.Errorf("provider not found: %s", name)
	}
	if !p.IsConfigured() {
		return nil, fmt.Errorf("provider not configured: %s", name)
	}
	return p, nil
}

// List returns all registered providers
func (r *Registry) List() []Provider {
	r.mu.RLock()
	defer r.mu.RUnlock()

	providers := make([]Provider, 0, len(r.providers))
	for _, p := range r.providers {
		providers = append(providers, p)
	}
	return providers
}

// ListConfigured returns only configured providers
func (r *Registry) ListConfigured() []Provider {
	r.mu.RLock()
	defer r.mu.RUnlock()

	providers := make([]Provider, 0)
	for _, p := range r.providers {
		if p.IsConfigured() {
			providers = append(providers, p)
		}
	}
	return providers
}

// AllModels returns models from all configured providers
func (r *Registry) AllModels() []ModelInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var models []ModelInfo
	for _, p := range r.providers {
		if p.IsConfigured() {
			models = append(models, p.Models()...)
		}
	}
	return models
}

// ProviderStatus returns the status of all providers
func (r *Registry) ProviderStatus() map[string]ProviderStatusInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	status := make(map[string]ProviderStatusInfo)
	for name, p := range r.providers {
		info := ProviderStatusInfo{
			Configured: p.IsConfigured(),
		}
		if p.IsConfigured() {
			models := p.Models()
			if len(models) > 0 {
				info.DefaultModel = models[0].ID
			}
		}
		status[name] = info
	}
	return status
}

// ProviderStatusInfo contains status information for a provider
type ProviderStatusInfo struct {
	Configured   bool   `json:"configured"`
	DefaultModel string `json:"default_model,omitempty"`
}
