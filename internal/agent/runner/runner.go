package runner

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/dezemandje/aule/internal/agent/llm"
	"github.com/dezemandje/aule/internal/agent/tool"
	"github.com/dezemandje/aule/internal/log"
)

var logger log.Logger

func init() {
	logger = log.GetLogger("agent", "runner")
}

const (
	// DefaultMaxIterations is the maximum number of agent loop iterations
	DefaultMaxIterations = 50

	// StopReasonStop indicates the LLM finished naturally
	StopReasonStop = "stop"

	// StopReasonToolUse indicates the LLM wants to use tools
	StopReasonToolUse = "tool_use"

	// StopReasonEndTurn indicates end of turn (OpenAI)
	StopReasonEndTurn = "end_turn"

	// StopReasonToolCalls indicates tool calls (OpenAI)
	StopReasonToolCalls = "tool_calls"
)

// Runner executes the agent loop
type Runner struct {
	provider      llm.Provider
	tools         *tool.Registry
	maxIterations int
	model         string
	temperature   float64
	maxTokens     int

	// Callbacks for progress reporting
	onToolCall   func(name string, input json.RawMessage)
	onToolResult func(name string, output string, isError bool)
	onText       func(text string)
	onIteration  func(iteration int, tokenUsage llm.TokenUsage)
}

// Config configures the runner
type Config struct {
	Provider      llm.Provider
	Tools         *tool.Registry
	MaxIterations int
	Model         string
	Temperature   float64
	MaxTokens     int
}

// NewRunner creates a new agent runner
func NewRunner(cfg Config) *Runner {
	maxIterations := cfg.MaxIterations
	if maxIterations <= 0 {
		maxIterations = DefaultMaxIterations
	}

	maxTokens := cfg.MaxTokens
	if maxTokens <= 0 {
		maxTokens = 4096
	}

	return &Runner{
		provider:      cfg.Provider,
		tools:         cfg.Tools,
		maxIterations: maxIterations,
		model:         cfg.Model,
		temperature:   cfg.Temperature,
		maxTokens:     maxTokens,
	}
}

// SetCallbacks sets progress callbacks
func (r *Runner) SetCallbacks(
	onToolCall func(name string, input json.RawMessage),
	onToolResult func(name string, output string, isError bool),
	onText func(text string),
	onIteration func(iteration int, tokenUsage llm.TokenUsage),
) {
	r.onToolCall = onToolCall
	r.onToolResult = onToolResult
	r.onText = onText
	r.onIteration = onIteration
}

// TaskInput contains the input for running a task
type TaskInput struct {
	SystemPrompt string
	UserPrompt   string
	Context      string
	WorkDir      string
	AllowedTools []string
}

// TaskResult contains the result of running a task
type TaskResult struct {
	Success    bool
	Result     string
	Error      string
	TokensUsed llm.TokenUsage
	Iterations int
	ToolCalls  []ToolCallRecord
}

// ToolCallRecord records a tool call
type ToolCallRecord struct {
	Name   string
	Input  string
	Output string
	Error  string
}

// Run executes the agent loop for a task
func (r *Runner) Run(ctx context.Context, input *TaskInput) (*TaskResult, error) {
	logger.Info("Starting agent run",
		"workDir", input.WorkDir,
		"allowedTools", input.AllowedTools,
	)

	// Build initial messages
	messages := []llm.Message{}

	// System message
	systemPrompt := input.SystemPrompt
	if input.Context != "" {
		systemPrompt += "\n\n## Context\n" + input.Context
	}
	messages = append(messages, llm.NewTextMessage("system", systemPrompt))

	// User message (task prompt)
	messages = append(messages, llm.NewTextMessage("user", input.UserPrompt))

	// Get tool definitions (filtered if specified)
	var toolDefs []llm.ToolDef
	if len(input.AllowedTools) > 0 {
		toolDefs = r.tools.FilteredToolDefs(input.AllowedTools)
	} else {
		toolDefs = r.tools.ToToolDefs()
	}

	logger.Debug("Tool definitions loaded", "count", len(toolDefs))

	// Track results
	result := &TaskResult{
		ToolCalls: make([]ToolCallRecord, 0),
	}

	// Agent loop
	for i := 0; i < r.maxIterations; i++ {
		result.Iterations = i + 1

		logger.Info("Agent iteration", "iteration", i+1, "maxIterations", r.maxIterations)

		// Call LLM
		resp, err := r.provider.Complete(ctx, &llm.CompletionRequest{
			Model:       r.model,
			Messages:    messages,
			Tools:       toolDefs,
			MaxTokens:   r.maxTokens,
			Temperature: r.temperature,
		})
		if err != nil {
			logger.Error("LLM call failed", "err", err)
			result.Error = fmt.Sprintf("LLM error: %v", err)
			return result, err
		}

		// Update token usage
		result.TokensUsed.InputTokens += resp.Usage.InputTokens
		result.TokensUsed.OutputTokens += resp.Usage.OutputTokens

		if r.onIteration != nil {
			r.onIteration(i+1, resp.Usage)
		}

		// Add assistant response to messages
		messages = append(messages, llm.Message{
			Role:    "assistant",
			Content: resp.Content,
		})

		// Extract and report text
		text := llm.ExtractText(resp.Content)
		if text != "" && r.onText != nil {
			r.onText(text)
		}

		logger.Debug("LLM response",
			"stopReason", resp.StopReason,
			"hasToolCalls", llm.HasToolCalls(resp.Content),
			"textLength", len(text),
		)

		// Check if we're done
		if resp.StopReason == StopReasonStop || resp.StopReason == StopReasonEndTurn {
			logger.Info("Agent completed", "reason", resp.StopReason, "iterations", i+1)
			result.Success = true
			result.Result = text
			return result, nil
		}

		// Handle tool calls
		if resp.StopReason == StopReasonToolUse || resp.StopReason == StopReasonToolCalls || llm.HasToolCalls(resp.Content) {
			toolCalls := llm.ExtractToolCalls(resp.Content)
			if len(toolCalls) == 0 {
				logger.Warn("Tool use indicated but no tool calls found")
				result.Success = true
				result.Result = text
				return result, nil
			}

			// Execute tools and collect results
			toolResults := make([]llm.ContentBlock, 0, len(toolCalls))

			for _, tc := range toolCalls {
				logger.Info("Executing tool", "name", tc.Name, "id", tc.ID)

				if r.onToolCall != nil {
					r.onToolCall(tc.Name, tc.Input)
				}

				// Execute tool
				output, err := r.tools.Execute(ctx, input.WorkDir, tc.Name, tc.Input)

				record := ToolCallRecord{
					Name:  tc.Name,
					Input: string(tc.Input),
				}

				if err != nil {
					logger.Warn("Tool execution failed", "name", tc.Name, "err", err)
					record.Error = err.Error()
					record.Output = err.Error()
					toolResults = append(toolResults, llm.NewToolResult(tc.ID, err.Error(), true))

					if r.onToolResult != nil {
						r.onToolResult(tc.Name, err.Error(), true)
					}
				} else {
					// Truncate very long outputs
					if len(output) > 50000 {
						output = output[:50000] + "\n... (output truncated)"
					}
					record.Output = output
					toolResults = append(toolResults, llm.NewToolResult(tc.ID, output, false))

					logger.Debug("Tool executed successfully", "name", tc.Name, "outputLength", len(output))

					if r.onToolResult != nil {
						r.onToolResult(tc.Name, output, false)
					}
				}

				result.ToolCalls = append(result.ToolCalls, record)
			}

			// Add tool results to messages
			// OpenAI expects each tool result as a separate message
			for _, tr := range toolResults {
				messages = append(messages, llm.Message{
					Role:    "tool",
					Content: []llm.ContentBlock{tr},
				})
			}
		}
	}

	// Exceeded max iterations
	logger.Warn("Max iterations reached", "maxIterations", r.maxIterations)
	result.Error = fmt.Sprintf("exceeded maximum iterations (%d)", r.maxIterations)
	result.Result = llm.ExtractText(messages[len(messages)-1].Content)
	return result, fmt.Errorf("max iterations exceeded")
}

// BuildSystemPrompt builds a system prompt from components
func BuildSystemPrompt(taskType, taskTitle, additionalInstructions string) string {
	var sb strings.Builder

	sb.WriteString("You are an AI agent working on a software development task.\n\n")

	if taskType != "" {
		sb.WriteString(fmt.Sprintf("Task Type: %s\n", taskType))
	}
	if taskTitle != "" {
		sb.WriteString(fmt.Sprintf("Task Title: %s\n", taskTitle))
	}

	sb.WriteString(`
You have access to tools to help you complete this task:
- read: Read file contents
- write: Write/create files  
- edit: Edit files with string replacement
- glob: Find files by pattern
- grep: Search file contents
- bash: Execute shell commands

Guidelines:
1. Start by understanding the task and exploring relevant code
2. Make small, focused changes
3. Test your changes when possible
4. Explain your reasoning as you work
5. Report your results when complete
`)

	if additionalInstructions != "" {
		sb.WriteString("\n## Additional Instructions\n")
		sb.WriteString(additionalInstructions)
	}

	return sb.String()
}
