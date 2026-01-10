package llm

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
)

// ProxyProvider implements the Provider interface by calling the LLM Proxy
type ProxyProvider struct {
	endpoint    string
	authToken   string
	provider    string
	model       string
	maxTokens   int
	temperature float64
	client      *http.Client
}

// ProxyConfig configures the proxy provider
type ProxyConfig struct {
	Endpoint    string
	AuthToken   string
	Provider    string
	Model       string
	MaxTokens   int
	Temperature float64
	Timeout     time.Duration
}

// NewProxyProvider creates a new proxy provider
func NewProxyProvider(cfg ProxyConfig) *ProxyProvider {
	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 10 * time.Minute
	}

	return &ProxyProvider{
		endpoint:    cfg.Endpoint,
		authToken:   cfg.AuthToken,
		provider:    cfg.Provider,
		model:       cfg.Model,
		maxTokens:   cfg.MaxTokens,
		temperature: cfg.Temperature,
		client: &http.Client{
			Timeout: timeout,
		},
	}
}

// Name returns the provider name
func (p *ProxyProvider) Name() string {
	return "proxy"
}

// Complete sends a non-streaming completion request to the proxy
func (p *ProxyProvider) Complete(ctx context.Context, req *CompletionRequest) (*CompletionResponse, error) {
	// Build proxy request body
	proxyReq := proxyCompleteRequest{
		Messages: req.Messages,
		Tools:    req.Tools,
	}

	body, err := json.Marshal(proxyReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	p.setHeaders(httpReq, req)

	// Send request
	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Check for errors
	if resp.StatusCode != http.StatusOK {
		var errorResp proxyErrorResponse
		if err := json.Unmarshal(respBody, &errorResp); err == nil && errorResp.Error != "" {
			return nil, fmt.Errorf("proxy error (%d): %s", resp.StatusCode, errorResp.Error)
		}
		return nil, fmt.Errorf("proxy error (%d): %s", resp.StatusCode, string(respBody))
	}

	// Parse response
	var proxyResp proxyCompleteResponse
	if err := json.Unmarshal(respBody, &proxyResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &CompletionResponse{
		Content:    proxyResp.Content,
		StopReason: proxyResp.StopReason,
		Usage:      proxyResp.Usage,
	}, nil
}

// CompleteStream sends a streaming completion request and processes events
func (p *ProxyProvider) CompleteStream(ctx context.Context, req *CompletionRequest, onEvent func(event StreamEvent)) (*CompletionResponse, error) {
	// Build proxy request body
	proxyReq := proxyCompleteRequest{
		Messages: req.Messages,
		Tools:    req.Tools,
	}

	body, err := json.Marshal(proxyReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	p.setHeaders(httpReq, req)
	httpReq.Header.Set("Accept", "text/event-stream")

	// Send request
	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Check for non-streaming error response
	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		var errorResp proxyErrorResponse
		if err := json.Unmarshal(respBody, &errorResp); err == nil && errorResp.Error != "" {
			return nil, fmt.Errorf("proxy error (%d): %s", resp.StatusCode, errorResp.Error)
		}
		return nil, fmt.Errorf("proxy error (%d): %s", resp.StatusCode, string(respBody))
	}

	// Accumulate content and track usage
	var content []ContentBlock
	var usage TokenUsage
	var stopReason string

	// Parse SSE stream
	scanner := bufio.NewScanner(resp.Body)
	var currentEvent string

	for scanner.Scan() {
		line := scanner.Text()

		// Parse event type
		if strings.HasPrefix(line, "event: ") {
			currentEvent = strings.TrimPrefix(line, "event: ")
			continue
		}

		// Parse data
		if strings.HasPrefix(line, "data: ") {
			data := strings.TrimPrefix(line, "data: ")

			switch currentEvent {
			case "start":
				var startData sseStartData
				if err := json.Unmarshal([]byte(data), &startData); err == nil {
					if onEvent != nil {
						onEvent(StreamEvent{Type: "start", Data: startData})
					}
				}

			case "content":
				var contentData sseContentData
				if err := json.Unmarshal([]byte(data), &contentData); err == nil {
					// Accumulate content
					block := ContentBlock{
						Type:  contentData.Type,
						Text:  contentData.Text,
						ID:    contentData.ID,
						Name:  contentData.Name,
						Input: contentData.Input,
					}

					// For text, we might need to merge with existing
					if contentData.Type == "text" {
						// Append to last text block or create new one
						if len(content) > 0 && content[len(content)-1].Type == "text" {
							content[len(content)-1].Text += contentData.Text
						} else {
							content = append(content, block)
						}
					} else {
						content = append(content, block)
					}

					if onEvent != nil {
						onEvent(StreamEvent{Type: "content", Data: contentData})
					}
				}

			case "usage":
				var usageData sseUsageData
				if err := json.Unmarshal([]byte(data), &usageData); err == nil {
					usage = TokenUsage{
						InputTokens:  usageData.InputTokens,
						OutputTokens: usageData.OutputTokens,
					}
					if onEvent != nil {
						onEvent(StreamEvent{Type: "usage", Data: usageData})
					}
				}

			case "done":
				var doneData sseDoneData
				if err := json.Unmarshal([]byte(data), &doneData); err == nil {
					stopReason = doneData.StopReason
					if onEvent != nil {
						onEvent(StreamEvent{Type: "done", Data: doneData})
					}
				}

			case "error":
				var errorData sseErrorData
				if err := json.Unmarshal([]byte(data), &errorData); err == nil {
					if onEvent != nil {
						onEvent(StreamEvent{Type: "error", Data: errorData})
					}
					return nil, fmt.Errorf("streaming error: %s", errorData.Error)
				}
			}

			currentEvent = ""
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading stream: %w", err)
	}

	return &CompletionResponse{
		Content:    content,
		StopReason: stopReason,
		Usage:      usage,
	}, nil
}

func (p *ProxyProvider) setHeaders(req *http.Request, compReq *CompletionRequest) {
	req.Header.Set("Content-Type", "application/json")

	if p.authToken != "" {
		req.Header.Set("Authorization", "Bearer "+p.authToken)
	}

	// Set LLM config headers
	provider := p.provider
	if provider == "" {
		provider = "openai"
	}
	req.Header.Set("X-LLM-Provider", provider)

	model := compReq.Model
	if model == "" {
		model = p.model
	}
	if model != "" {
		req.Header.Set("X-LLM-Model", model)
	}

	maxTokens := compReq.MaxTokens
	if maxTokens == 0 {
		maxTokens = p.maxTokens
	}
	if maxTokens > 0 {
		req.Header.Set("X-LLM-Max-Tokens", fmt.Sprintf("%d", maxTokens))
	}

	temp := compReq.Temperature
	if temp == 0 {
		temp = p.temperature
	}
	req.Header.Set("X-LLM-Temperature", fmt.Sprintf("%.1f", temp))
}

// StreamEvent represents a streaming event
type StreamEvent struct {
	Type string
	Data interface{}
}

// Proxy request/response types

type proxyCompleteRequest struct {
	Messages []Message `json:"messages"`
	Tools    []ToolDef `json:"tools,omitempty"`
}

type proxyCompleteResponse struct {
	Content    []ContentBlock `json:"content"`
	StopReason string         `json:"stop_reason"`
	Usage      TokenUsage     `json:"usage"`
	Model      string         `json:"model"`
}

type proxyErrorResponse struct {
	Error string `json:"error"`
}

// SSE data types

type sseStartData struct {
	Model string `json:"model"`
}

type sseContentData struct {
	Type  string          `json:"type"`
	Text  string          `json:"text,omitempty"`
	ID    string          `json:"id,omitempty"`
	Name  string          `json:"name,omitempty"`
	Input json.RawMessage `json:"input,omitempty"`
}

type sseUsageData struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

type sseDoneData struct {
	StopReason string `json:"stop_reason"`
}

type sseErrorData struct {
	Error string `json:"error"`
}
