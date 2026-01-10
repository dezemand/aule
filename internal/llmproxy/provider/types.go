package provider

import "encoding/json"

// Message represents a chat message
type Message struct {
	Role    string         `json:"role"`
	Content []ContentBlock `json:"content"`
}

// ContentBlock represents a content block within a message
type ContentBlock struct {
	Type string `json:"type"`

	// For type="text"
	Text string `json:"text,omitempty"`

	// For type="tool_use"
	ID    string          `json:"id,omitempty"`
	Name  string          `json:"name,omitempty"`
	Input json.RawMessage `json:"input,omitempty"`

	// For type="tool_result"
	ToolUseID string `json:"tool_use_id,omitempty"`
	Content   string `json:"content,omitempty"`
	IsError   bool   `json:"is_error,omitempty"`
}

// ToolDef defines a tool for the LLM
type ToolDef struct {
	Type     string       `json:"type"`
	Function ToolFunction `json:"function"`
}

// ToolFunction defines the function details
type ToolFunction struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters"`
}

// CompleteRequest is a request to the LLM provider
type CompleteRequest struct {
	Model       string
	MaxTokens   int
	Temperature float64
	Messages    []Message
	Tools       []ToolDef
}

// CompleteResponse is a response from the LLM provider
type CompleteResponse struct {
	Content    []ContentBlock `json:"content"`
	StopReason string         `json:"stop_reason"`
	Usage      TokenUsage     `json:"usage"`
	Model      string         `json:"model"`
}

// TokenUsage tracks token consumption
type TokenUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

// StreamEvent represents a streaming event from the provider
type StreamEvent struct {
	Type string      `json:"type"` // "start", "content", "usage", "done", "error"
	Data interface{} `json:"data"`
}

// StreamStartData is sent at the start of streaming
type StreamStartData struct {
	Model string `json:"model"`
}

// StreamContentData is sent for content chunks
type StreamContentData struct {
	ContentBlock
}

// StreamUsageData is sent with token usage
type StreamUsageData struct {
	TokenUsage
}

// StreamDoneData is sent when streaming completes
type StreamDoneData struct {
	StopReason string `json:"stop_reason"`
}

// StreamErrorData is sent on error
type StreamErrorData struct {
	Error string `json:"error"`
}

// ModelInfo describes an available model
type ModelInfo struct {
	ID       string `json:"id"`
	Provider string `json:"provider"`
	Name     string `json:"name"`
}
