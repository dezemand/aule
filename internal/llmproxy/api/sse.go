package api

import (
	"encoding/json"
	"fmt"

	"github.com/gofiber/fiber/v2"
)

// SSEWriter handles Server-Sent Events writing to a Fiber context
type SSEWriter struct {
	ctx *fiber.Ctx
}

// NewSSEWriter creates a new SSE writer
func NewSSEWriter(c *fiber.Ctx) *SSEWriter {
	// Set SSE headers
	c.Set("Content-Type", "text/event-stream")
	c.Set("Cache-Control", "no-cache")
	c.Set("Connection", "keep-alive")
	c.Set("Transfer-Encoding", "chunked")

	return &SSEWriter{ctx: c}
}

// WriteEvent writes an SSE event with the given type and data
func (w *SSEWriter) WriteEvent(eventType string, data interface{}) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal event data: %w", err)
	}

	// Format: "event: <type>\ndata: <json>\n\n"
	event := fmt.Sprintf("event: %s\ndata: %s\n\n", eventType, string(jsonData))

	if _, err := w.ctx.WriteString(event); err != nil {
		return fmt.Errorf("failed to write event: %w", err)
	}

	return nil
}

// WriteRawEvent writes a raw SSE event string
func (w *SSEWriter) WriteRawEvent(eventType string, data string) error {
	event := fmt.Sprintf("event: %s\ndata: %s\n\n", eventType, data)

	if _, err := w.ctx.WriteString(event); err != nil {
		return fmt.Errorf("failed to write event: %w", err)
	}

	return nil
}

// WriteComment writes an SSE comment (for keepalive)
func (w *SSEWriter) WriteComment(comment string) error {
	if _, err := w.ctx.WriteString(": " + comment + "\n\n"); err != nil {
		return fmt.Errorf("failed to write comment: %w", err)
	}

	return nil
}

// Flush flushes the response writer (if supported)
// Note: Fiber handles flushing automatically in most cases
func (w *SSEWriter) Flush() {
	// Fiber's response writer handles flushing
	// This is a no-op but kept for interface compatibility
}

// SSE event types used by the proxy
const (
	SSEEventStart   = "start"
	SSEEventContent = "content"
	SSEEventUsage   = "usage"
	SSEEventDone    = "done"
	SSEEventError   = "error"
)

// SSE data types for JSON serialization

// SSEStartData is sent at stream start
type SSEStartData struct {
	Model string `json:"model"`
}

// SSEContentData is sent for content chunks
type SSEContentData struct {
	Type string `json:"type"`

	// For text
	Text string `json:"text,omitempty"`

	// For tool_use
	ID    string          `json:"id,omitempty"`
	Name  string          `json:"name,omitempty"`
	Input json.RawMessage `json:"input,omitempty"`
}

// SSEUsageData is sent with token usage
type SSEUsageData struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

// SSEDoneData is sent when streaming completes
type SSEDoneData struct {
	StopReason string `json:"stop_reason"`
}

// SSEErrorData is sent on error
type SSEErrorData struct {
	Error string `json:"error"`
}
