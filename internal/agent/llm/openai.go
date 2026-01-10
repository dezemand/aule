package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	defaultOpenAIBaseURL = "https://api.openai.com/v1"
	defaultModel         = "gpt-4o"
	defaultMaxTokens     = 4096
	defaultTimeout       = 5 * time.Minute
)

// OpenAIProvider implements the Provider interface for OpenAI-compatible APIs
type OpenAIProvider struct {
	baseURL string
	apiKey  string
	model   string
	client  *http.Client
}

// OpenAIConfig configures the OpenAI provider
type OpenAIConfig struct {
	BaseURL string
	APIKey  string
	Model   string
	Timeout time.Duration
}

// NewOpenAIProvider creates a new OpenAI-compatible provider
func NewOpenAIProvider(cfg OpenAIConfig) *OpenAIProvider {
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = defaultOpenAIBaseURL
	}

	model := cfg.Model
	if model == "" {
		model = defaultModel
	}

	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = defaultTimeout
	}

	return &OpenAIProvider{
		baseURL: baseURL,
		apiKey:  cfg.APIKey,
		model:   model,
		client: &http.Client{
			Timeout: timeout,
		},
	}
}

// Name returns the provider name
func (p *OpenAIProvider) Name() string {
	return "openai"
}

// Complete sends a completion request to the OpenAI API
func (p *OpenAIProvider) Complete(ctx context.Context, req *CompletionRequest) (*CompletionResponse, error) {
	// Convert to OpenAI API format
	openaiReq := p.buildRequest(req)

	// Serialize request
	body, err := json.Marshal(openaiReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)

	// Send request
	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Check for errors
	if resp.StatusCode != http.StatusOK {
		var errorResp openaiErrorResponse
		if err := json.Unmarshal(respBody, &errorResp); err == nil && errorResp.Error.Message != "" {
			return nil, fmt.Errorf("API error (%d): %s", resp.StatusCode, errorResp.Error.Message)
		}
		return nil, fmt.Errorf("API error (%d): %s", resp.StatusCode, string(respBody))
	}

	// Parse response
	var openaiResp openaiChatResponse
	if err := json.Unmarshal(respBody, &openaiResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Convert to our format
	return p.parseResponse(&openaiResp), nil
}

// buildRequest converts our request format to OpenAI format
func (p *OpenAIProvider) buildRequest(req *CompletionRequest) *openaiChatRequest {
	model := req.Model
	if model == "" {
		model = p.model
	}

	maxTokens := req.MaxTokens
	if maxTokens == 0 {
		maxTokens = defaultMaxTokens
	}

	// Convert messages
	messages := make([]openaiMessage, 0, len(req.Messages))
	for _, msg := range req.Messages {
		messages = append(messages, p.convertMessage(msg))
	}

	// Convert tools
	var tools []openaiTool
	if len(req.Tools) > 0 {
		tools = make([]openaiTool, len(req.Tools))
		for i, t := range req.Tools {
			tools[i] = openaiTool{
				Type: "function",
				Function: openaiFunction{
					Name:        t.Function.Name,
					Description: t.Function.Description,
					Parameters:  t.Function.Parameters,
				},
			}
		}
	}

	return &openaiChatRequest{
		Model:       model,
		Messages:    messages,
		MaxTokens:   maxTokens,
		Temperature: req.Temperature,
		Tools:       tools,
	}
}

// convertMessage converts our message format to OpenAI format
func (p *OpenAIProvider) convertMessage(msg Message) openaiMessage {
	// Handle tool results specially
	if msg.Role == "tool" {
		// OpenAI expects tool results as separate messages
		// For simplicity, combine them into a single message
		// In practice, you'd send multiple messages
		if len(msg.Content) > 0 && msg.Content[0].Type == "tool_result" {
			return openaiMessage{
				Role:       "tool",
				Content:    msg.Content[0].Content,
				ToolCallID: msg.Content[0].ToolUseID,
			}
		}
	}

	// Handle assistant messages with tool calls
	if msg.Role == "assistant" && HasToolCalls(msg.Content) {
		toolCalls := make([]openaiToolCall, 0)
		var textContent string

		for _, block := range msg.Content {
			if block.Type == "tool_use" {
				toolCalls = append(toolCalls, openaiToolCall{
					ID:   block.ID,
					Type: "function",
					Function: openaiToolCallFunction{
						Name:      block.Name,
						Arguments: string(block.Input),
					},
				})
			} else if block.Type == "text" {
				textContent += block.Text
			}
		}

		return openaiMessage{
			Role:      "assistant",
			Content:   textContent,
			ToolCalls: toolCalls,
		}
	}

	// Simple text message
	return openaiMessage{
		Role:    msg.Role,
		Content: ExtractText(msg.Content),
	}
}

// parseResponse converts OpenAI response to our format
func (p *OpenAIProvider) parseResponse(resp *openaiChatResponse) *CompletionResponse {
	if len(resp.Choices) == 0 {
		return &CompletionResponse{
			StopReason: "error",
		}
	}

	choice := resp.Choices[0]
	content := make([]ContentBlock, 0)

	// Add text content if present
	if choice.Message.Content != "" {
		content = append(content, ContentBlock{
			Type: "text",
			Text: choice.Message.Content,
		})
	}

	// Add tool calls if present
	for _, tc := range choice.Message.ToolCalls {
		content = append(content, ContentBlock{
			Type:  "tool_use",
			ID:    tc.ID,
			Name:  tc.Function.Name,
			Input: json.RawMessage(tc.Function.Arguments),
		})
	}

	// Map finish reason
	stopReason := choice.FinishReason
	if stopReason == "tool_calls" {
		stopReason = "tool_use"
	}

	return &CompletionResponse{
		Content:    content,
		StopReason: stopReason,
		Usage: TokenUsage{
			InputTokens:  resp.Usage.PromptTokens,
			OutputTokens: resp.Usage.CompletionTokens,
		},
	}
}

// OpenAI API types

type openaiChatRequest struct {
	Model       string          `json:"model"`
	Messages    []openaiMessage `json:"messages"`
	MaxTokens   int             `json:"max_tokens,omitempty"`
	Temperature float64         `json:"temperature,omitempty"`
	Tools       []openaiTool    `json:"tools,omitempty"`
}

type openaiMessage struct {
	Role       string           `json:"role"`
	Content    string           `json:"content,omitempty"`
	ToolCalls  []openaiToolCall `json:"tool_calls,omitempty"`
	ToolCallID string           `json:"tool_call_id,omitempty"`
}

type openaiToolCall struct {
	ID       string                 `json:"id"`
	Type     string                 `json:"type"`
	Function openaiToolCallFunction `json:"function"`
}

type openaiToolCallFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type openaiTool struct {
	Type     string         `json:"type"`
	Function openaiFunction `json:"function"`
}

type openaiFunction struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters"`
}

type openaiChatResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index        int           `json:"index"`
		Message      openaiMessage `json:"message"`
		FinishReason string        `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

type openaiErrorResponse struct {
	Error struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    string `json:"code"`
	} `json:"error"`
}
