package llm

import "context"

// Provider is the interface for LLM providers
type Provider interface {
	// Complete sends a completion request and returns the response
	Complete(ctx context.Context, req *CompletionRequest) (*CompletionResponse, error)

	// Name returns the provider name
	Name() string
}
