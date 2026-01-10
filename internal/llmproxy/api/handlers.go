package api

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/dezemandje/aule/internal/llmproxy/config"
	"github.com/dezemandje/aule/internal/llmproxy/provider"
	"github.com/dezemandje/aule/internal/log"
	"github.com/gofiber/fiber/v2"
	"github.com/valyala/fasthttp"
)

var logger log.Logger

func init() {
	logger = log.GetLogger("llmproxy", "api")
}

// Handler holds dependencies for API handlers
type Handler struct {
	registry *provider.Registry
	config   *config.Config
}

// NewHandler creates a new API handler
func NewHandler(registry *provider.Registry, cfg *config.Config) *Handler {
	return &Handler{
		registry: registry,
		config:   cfg,
	}
}

// CompleteRequest is the request body for /v1/complete
type CompleteRequest struct {
	Messages []provider.Message `json:"messages"`
	Tools    []provider.ToolDef `json:"tools,omitempty"`
}

// CompleteHandler handles POST /v1/complete
func (h *Handler) CompleteHandler(c *fiber.Ctx) error {
	// Get LLM config from middleware
	llmCfg := GetLLMConfig(c)

	// Parse request body
	var req CompleteRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body: " + err.Error(),
		})
	}

	// Validate messages
	if len(req.Messages) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "messages array is required",
		})
	}

	// Get provider
	prov, err := h.registry.Get(llmCfg.Provider)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	// Build provider request
	provReq := &provider.CompleteRequest{
		Model:       llmCfg.Model,
		MaxTokens:   llmCfg.MaxTokens,
		Temperature: llmCfg.Temperature,
		Messages:    req.Messages,
		Tools:       req.Tools,
	}

	// Check if streaming is requested
	acceptHeader := c.Get("Accept")
	wantsStreaming := strings.Contains(acceptHeader, "text/event-stream")

	agentID := GetAgentID(c)
	logger.Info("LLM completion request",
		"agent_id", agentID,
		"provider", llmCfg.Provider,
		"model", llmCfg.Model,
		"streaming", wantsStreaming,
		"messages", len(req.Messages),
		"tools", len(req.Tools),
	)

	if wantsStreaming {
		return h.handleStreaming(c, prov, provReq)
	}
	return h.handleNonStreaming(c, prov, provReq)
}

// handleNonStreaming handles non-streaming completion
func (h *Handler) handleNonStreaming(c *fiber.Ctx, prov provider.Provider, req *provider.CompleteRequest) error {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(c.Context(), h.config.Limits.RequestTimeout)
	defer cancel()

	startTime := time.Now()

	// Call provider
	resp, err := prov.Complete(ctx, req)
	if err != nil {
		logger.Error("LLM completion failed", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	duration := time.Since(startTime)
	logger.Info("LLM completion completed",
		"duration", duration,
		"input_tokens", resp.Usage.InputTokens,
		"output_tokens", resp.Usage.OutputTokens,
		"stop_reason", resp.StopReason,
	)

	return c.JSON(resp)
}

// handleStreaming handles streaming completion with SSE
func (h *Handler) handleStreaming(c *fiber.Ctx, prov provider.Provider, req *provider.CompleteRequest) error {
	// Create context with streaming timeout
	ctx, cancel := context.WithTimeout(context.Background(), h.config.Limits.StreamingTimeout)

	startTime := time.Now()

	// Start streaming from provider
	events, err := prov.Stream(ctx, req)
	if err != nil {
		cancel()
		logger.Error("LLM streaming failed to start", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	// Set SSE headers
	c.Set("Content-Type", "text/event-stream")
	c.Set("Cache-Control", "no-cache")
	c.Set("Connection", "keep-alive")

	// Track metrics
	var totalInputTokens, totalOutputTokens int
	var stopReason string

	// Use fasthttp's streaming
	c.Context().SetBodyStreamWriter(func(w *bufio.Writer) {
		defer cancel()

		for event := range events {
			var eventData []byte
			var eventType string

			switch event.Type {
			case "start":
				eventType = SSEEventStart
				if data, ok := event.Data.(provider.StreamStartData); ok {
					eventData, _ = json.Marshal(SSEStartData{Model: data.Model})
				}

			case "content":
				eventType = SSEEventContent
				if data, ok := event.Data.(provider.StreamContentData); ok {
					eventData, _ = json.Marshal(SSEContentData{
						Type:  data.Type,
						Text:  data.Text,
						ID:    data.ID,
						Name:  data.Name,
						Input: data.Input,
					})
				}

			case "usage":
				eventType = SSEEventUsage
				if data, ok := event.Data.(provider.StreamUsageData); ok {
					totalInputTokens = data.InputTokens
					totalOutputTokens = data.OutputTokens
					eventData, _ = json.Marshal(SSEUsageData{
						InputTokens:  data.InputTokens,
						OutputTokens: data.OutputTokens,
					})
				}

			case "done":
				eventType = SSEEventDone
				if data, ok := event.Data.(provider.StreamDoneData); ok {
					stopReason = data.StopReason
					eventData, _ = json.Marshal(SSEDoneData{StopReason: data.StopReason})
				}

			case "error":
				eventType = SSEEventError
				if data, ok := event.Data.(provider.StreamErrorData); ok {
					eventData, _ = json.Marshal(SSEErrorData{Error: data.Error})
				}
			}

			if eventType != "" && eventData != nil {
				fmt.Fprintf(w, "event: %s\ndata: %s\n\n", eventType, string(eventData))
				w.Flush()
			}
		}

		duration := time.Since(startTime)
		logger.Info("LLM streaming completed",
			"duration", duration,
			"input_tokens", totalInputTokens,
			"output_tokens", totalOutputTokens,
			"stop_reason", stopReason,
		)
	})

	return nil
}

// ModelsHandler handles GET /v1/models
func (h *Handler) ModelsHandler(c *fiber.Ctx) error {
	models := h.registry.AllModels()
	return c.JSON(fiber.Map{
		"models": models,
	})
}

// HealthHandler handles GET /v1/health
func (h *Handler) HealthHandler(c *fiber.Ctx) error {
	providers := h.registry.ProviderStatus()

	return c.JSON(fiber.Map{
		"status":    "healthy",
		"providers": providers,
	})
}

// StreamWriterFunc is the signature for SetBodyStreamWriter
type StreamWriterFunc func(w *bufio.Writer)

// SetBodyStreamWriter sets the body stream writer on the fasthttp context
func SetBodyStreamWriter(c *fiber.Ctx, fn StreamWriterFunc) {
	c.Context().SetBodyStreamWriter(fasthttp.StreamWriter(fn))
}
