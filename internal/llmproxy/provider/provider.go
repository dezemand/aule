package provider

import "context"

// Provider is the interface for LLM providers
type Provider interface {
	// Name returns the provider identifier
	Name() string

	// Complete sends a completion request and returns the full response
	Complete(ctx context.Context, req *CompleteRequest) (*CompleteResponse, error)

	// Stream sends a completion request and streams the response
	Stream(ctx context.Context, req *CompleteRequest) (<-chan StreamEvent, error)

	// Models returns available models for this provider
	Models() []ModelInfo

	// IsConfigured returns true if the provider is properly configured
	IsConfigured() bool
}
