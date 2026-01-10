package llm

import "encoding/json"

// Message represents a chat message
type Message struct {
	Role    string         `json:"role"` // "system", "user", "assistant", "tool"
	Content []ContentBlock `json:"content"`
}

// ContentBlock represents a content block within a message
type ContentBlock struct {
	Type string `json:"type"` // "text", "tool_use", "tool_result"

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
	Type     string       `json:"type"` // "function"
	Function ToolFunction `json:"function"`
}

// ToolFunction defines the function details
type ToolFunction struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters"`
}

// CompletionRequest is a request to the LLM
type CompletionRequest struct {
	Model       string
	Messages    []Message
	Tools       []ToolDef
	MaxTokens   int
	Temperature float64
}

// CompletionResponse is a response from the LLM
type CompletionResponse struct {
	Content    []ContentBlock
	StopReason string
	Usage      TokenUsage
}

// TokenUsage tracks token consumption
type TokenUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

// NewTextMessage creates a message with text content
func NewTextMessage(role, text string) Message {
	return Message{
		Role: role,
		Content: []ContentBlock{
			{Type: "text", Text: text},
		},
	}
}

// NewToolResultMessage creates a message with tool results
func NewToolResultMessage(results []ContentBlock) Message {
	return Message{
		Role:    "tool",
		Content: results,
	}
}

// NewToolResult creates a tool result content block
func NewToolResult(toolUseID, content string, isError bool) ContentBlock {
	return ContentBlock{
		Type:      "tool_result",
		ToolUseID: toolUseID,
		Content:   content,
		IsError:   isError,
	}
}

// ExtractText extracts text content from content blocks
func ExtractText(blocks []ContentBlock) string {
	var result string
	for _, block := range blocks {
		if block.Type == "text" {
			result += block.Text
		}
	}
	return result
}

// ExtractToolCalls extracts tool use blocks from content blocks
func ExtractToolCalls(blocks []ContentBlock) []ContentBlock {
	var calls []ContentBlock
	for _, block := range blocks {
		if block.Type == "tool_use" {
			calls = append(calls, block)
		}
	}
	return calls
}

// HasToolCalls checks if there are any tool calls in the content
func HasToolCalls(blocks []ContentBlock) bool {
	for _, block := range blocks {
		if block.Type == "tool_use" {
			return true
		}
	}
	return false
}
