package provider

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/dezemandje/aule/internal/llmproxy/config"
)

const (
	openaiProviderName = "openai"
)

// OpenAIProvider implements the Provider interface for OpenAI-compatible APIs
type OpenAIProvider struct {
	apiKey       string
	baseURL      string
	defaultModel string
	client       *http.Client
}

// NewOpenAIProvider creates a new OpenAI provider
func NewOpenAIProvider(cfg config.OpenAIConfig) *OpenAIProvider {
	return &OpenAIProvider{
		apiKey:       cfg.APIKey,
		baseURL:      cfg.BaseURL,
		defaultModel: cfg.DefaultModel,
		client: &http.Client{
			Timeout: 10 * time.Minute, // Long timeout for streaming
		},
	}
}

// Name returns the provider name
func (p *OpenAIProvider) Name() string {
	return openaiProviderName
}

// IsConfigured returns true if the provider has an API key
func (p *OpenAIProvider) IsConfigured() bool {
	return p.apiKey != ""
}

// Models returns available OpenAI models
func (p *OpenAIProvider) Models() []ModelInfo {
	return []ModelInfo{
		{ID: "gpt-4o", Provider: openaiProviderName, Name: "GPT-4o"},
		{ID: "gpt-4o-mini", Provider: openaiProviderName, Name: "GPT-4o Mini"},
		{ID: "gpt-4-turbo", Provider: openaiProviderName, Name: "GPT-4 Turbo"},
		{ID: "gpt-4", Provider: openaiProviderName, Name: "GPT-4"},
		{ID: "gpt-3.5-turbo", Provider: openaiProviderName, Name: "GPT-3.5 Turbo"},
	}
}

// Complete sends a non-streaming completion request
func (p *OpenAIProvider) Complete(ctx context.Context, req *CompleteRequest) (*CompleteResponse, error) {
	model := req.Model
	if model == "" {
		model = p.defaultModel
	}

	openaiReq := p.buildRequest(req, model, false)

	body, err := json.Marshal(openaiReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	p.setHeaders(httpReq)

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var errorResp openaiErrorResponse
		if err := json.Unmarshal(respBody, &errorResp); err == nil && errorResp.Error.Message != "" {
			return nil, fmt.Errorf("OpenAI API error (%d): %s", resp.StatusCode, errorResp.Error.Message)
		}
		return nil, fmt.Errorf("OpenAI API error (%d): %s", resp.StatusCode, string(respBody))
	}

	var openaiResp openaiChatResponse
	if err := json.Unmarshal(respBody, &openaiResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return p.parseResponse(&openaiResp, model), nil
}

// Stream sends a streaming completion request
func (p *OpenAIProvider) Stream(ctx context.Context, req *CompleteRequest) (<-chan StreamEvent, error) {
	model := req.Model
	if model == "" {
		model = p.defaultModel
	}

	openaiReq := p.buildRequest(req, model, true)

	body, err := json.Marshal(openaiReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	p.setHeaders(httpReq)

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		respBody, _ := io.ReadAll(resp.Body)
		var errorResp openaiErrorResponse
		if err := json.Unmarshal(respBody, &errorResp); err == nil && errorResp.Error.Message != "" {
			return nil, fmt.Errorf("OpenAI API error (%d): %s", resp.StatusCode, errorResp.Error.Message)
		}
		return nil, fmt.Errorf("OpenAI API error (%d): %s", resp.StatusCode, string(respBody))
	}

	events := make(chan StreamEvent, 100)

	go func() {
		defer close(events)
		defer resp.Body.Close()

		// Send start event
		events <- StreamEvent{
			Type: "start",
			Data: StreamStartData{Model: model},
		}

		// Track accumulated content for tool calls
		var currentToolCall *openaiToolCallDelta
		var usage TokenUsage
		var stopReason string

		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			line := scanner.Text()

			// Skip empty lines and comments
			if line == "" || strings.HasPrefix(line, ":") {
				continue
			}

			// Parse SSE data line
			if !strings.HasPrefix(line, "data: ") {
				continue
			}

			data := strings.TrimPrefix(line, "data: ")

			// Check for end of stream
			if data == "[DONE]" {
				break
			}

			var chunk openaiStreamChunk
			if err := json.Unmarshal([]byte(data), &chunk); err != nil {
				continue // Skip malformed chunks
			}

			if len(chunk.Choices) == 0 {
				// Check for usage in final message
				if chunk.Usage.TotalTokens > 0 {
					usage = TokenUsage{
						InputTokens:  chunk.Usage.PromptTokens,
						OutputTokens: chunk.Usage.CompletionTokens,
					}
				}
				continue
			}

			choice := chunk.Choices[0]

			// Update stop reason
			if choice.FinishReason != "" {
				stopReason = choice.FinishReason
				if stopReason == "tool_calls" {
					stopReason = "tool_use"
				}
			}

			// Handle text content
			if choice.Delta.Content != "" {
				events <- StreamEvent{
					Type: "content",
					Data: StreamContentData{
						ContentBlock: ContentBlock{
							Type: "text",
							Text: choice.Delta.Content,
						},
					},
				}
			}

			// Handle tool calls
			for _, tc := range choice.Delta.ToolCalls {
				if tc.ID != "" {
					// New tool call starting
					if currentToolCall != nil && currentToolCall.ID != "" {
						// Emit previous tool call
						events <- StreamEvent{
							Type: "content",
							Data: StreamContentData{
								ContentBlock: ContentBlock{
									Type:  "tool_use",
									ID:    currentToolCall.ID,
									Name:  currentToolCall.Function.Name,
									Input: json.RawMessage(currentToolCall.Function.Arguments),
								},
							},
						}
					}
					currentToolCall = &openaiToolCallDelta{
						ID:       tc.ID,
						Type:     tc.Type,
						Function: tc.Function,
					}
				} else if currentToolCall != nil {
					// Continuation of current tool call
					currentToolCall.Function.Arguments += tc.Function.Arguments
				}
			}
		}

		// Emit final tool call if any
		if currentToolCall != nil && currentToolCall.ID != "" {
			events <- StreamEvent{
				Type: "content",
				Data: StreamContentData{
					ContentBlock: ContentBlock{
						Type:  "tool_use",
						ID:    currentToolCall.ID,
						Name:  currentToolCall.Function.Name,
						Input: json.RawMessage(currentToolCall.Function.Arguments),
					},
				},
			}
		}

		// Send usage event
		if usage.InputTokens > 0 || usage.OutputTokens > 0 {
			events <- StreamEvent{
				Type: "usage",
				Data: StreamUsageData{TokenUsage: usage},
			}
		}

		// Send done event
		events <- StreamEvent{
			Type: "done",
			Data: StreamDoneData{StopReason: stopReason},
		}
	}()

	return events, nil
}

func (p *OpenAIProvider) setHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.apiKey)
}

func (p *OpenAIProvider) buildRequest(req *CompleteRequest, model string, stream bool) *openaiChatRequest {
	maxTokens := req.MaxTokens
	if maxTokens == 0 {
		maxTokens = 4096
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

	openaiReq := &openaiChatRequest{
		Model:       model,
		Messages:    messages,
		MaxTokens:   maxTokens,
		Temperature: req.Temperature,
		Tools:       tools,
		Stream:      stream,
	}

	if stream {
		// Request usage in streaming mode
		openaiReq.StreamOptions = &openaiStreamOptions{
			IncludeUsage: true,
		}
	}

	return openaiReq
}

func (p *OpenAIProvider) convertMessage(msg Message) openaiMessage {
	// Handle tool results
	if msg.Role == "tool" {
		if len(msg.Content) > 0 && msg.Content[0].Type == "tool_result" {
			return openaiMessage{
				Role:       "tool",
				Content:    msg.Content[0].Content,
				ToolCallID: msg.Content[0].ToolUseID,
			}
		}
	}

	// Handle assistant messages with tool calls
	if msg.Role == "assistant" && hasToolCalls(msg.Content) {
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
		Content: extractText(msg.Content),
	}
}

func (p *OpenAIProvider) parseResponse(resp *openaiChatResponse, model string) *CompleteResponse {
	if len(resp.Choices) == 0 {
		return &CompleteResponse{
			StopReason: "error",
			Model:      model,
		}
	}

	choice := resp.Choices[0]
	content := make([]ContentBlock, 0)

	if choice.Message.Content != "" {
		content = append(content, ContentBlock{
			Type: "text",
			Text: choice.Message.Content,
		})
	}

	for _, tc := range choice.Message.ToolCalls {
		content = append(content, ContentBlock{
			Type:  "tool_use",
			ID:    tc.ID,
			Name:  tc.Function.Name,
			Input: json.RawMessage(tc.Function.Arguments),
		})
	}

	stopReason := choice.FinishReason
	if stopReason == "tool_calls" {
		stopReason = "tool_use"
	}

	return &CompleteResponse{
		Content:    content,
		StopReason: stopReason,
		Usage: TokenUsage{
			InputTokens:  resp.Usage.PromptTokens,
			OutputTokens: resp.Usage.CompletionTokens,
		},
		Model: model,
	}
}

// Helper functions

func hasToolCalls(blocks []ContentBlock) bool {
	for _, block := range blocks {
		if block.Type == "tool_use" {
			return true
		}
	}
	return false
}

func extractText(blocks []ContentBlock) string {
	var result string
	for _, block := range blocks {
		if block.Type == "text" {
			result += block.Text
		}
	}
	return result
}

// OpenAI API types

type openaiChatRequest struct {
	Model         string               `json:"model"`
	Messages      []openaiMessage      `json:"messages"`
	MaxTokens     int                  `json:"max_tokens,omitempty"`
	Temperature   float64              `json:"temperature,omitempty"`
	Tools         []openaiTool         `json:"tools,omitempty"`
	Stream        bool                 `json:"stream,omitempty"`
	StreamOptions *openaiStreamOptions `json:"stream_options,omitempty"`
}

type openaiStreamOptions struct {
	IncludeUsage bool `json:"include_usage,omitempty"`
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

type openaiStreamChunk struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index int `json:"index"`
		Delta struct {
			Role      string                `json:"role,omitempty"`
			Content   string                `json:"content,omitempty"`
			ToolCalls []openaiToolCallDelta `json:"tool_calls,omitempty"`
		} `json:"delta"`
		FinishReason string `json:"finish_reason,omitempty"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage,omitempty"`
}

type openaiToolCallDelta struct {
	Index    int    `json:"index"`
	ID       string `json:"id,omitempty"`
	Type     string `json:"type,omitempty"`
	Function struct {
		Name      string `json:"name,omitempty"`
		Arguments string `json:"arguments,omitempty"`
	} `json:"function"`
}

type openaiErrorResponse struct {
	Error struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    string `json:"code"`
	} `json:"error"`
}
